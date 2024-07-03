package luma

import (
	"container/list"
	"fmt"
	"net/netip"

	"github.com/lumavpn/luma/component/resolver"
	"github.com/lumavpn/luma/component/trie"
	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/adapter"
	"github.com/lumavpn/luma/proxy/outbound"
	"github.com/lumavpn/luma/proxy/provider"
	"github.com/lumavpn/luma/tunnel"
)

// parseConfig is used to parse and load the general configuration used by Luma
func (lu *Luma) parseConfig(cfg *config.Config) error {
	proxies, providers, err := parseProxies(cfg)
	if err != nil {
		return err
	}
	lu.SetProxies(proxies)
	lu.SetProxyProviders(providers)
	log.Debugf("Have %d proxies", len(proxies))

	lu.proxyDialer.UpdateProxies(proxies, providers)

	ruleProviders, err := parseRuleProviders(cfg.RuleProviders)
	if err != nil {
		return err
	}
	lu.SetRuleProviders(ruleProviders)
	subRules, err := parseSubRules(cfg.SubRules, proxies)
	if err != nil {
		return err
	}

	rules, err := parseRules(cfg.Rules, proxies, subRules, "rules")
	if err != nil {
		return err
	}

	lu.SetRules(rules)
	lu.SetSubRules(subRules)

	log.Debugf("Have %d rules", len(rules))
	hosts, err := parseHosts(cfg.Hosts)
	if err != nil {
		return err
	}
	lu.SetHosts(hosts)

	cfg.DNS, err = parseDNS(cfg, lu.tunnel, hosts, rules, ruleProviders)
	if err != nil {
		return err
	}

	if err := configureTun(cfg, lu.tunnel); err != nil {
		log.Fatalf("unable to parse tun config: %v", err)
	}

	return nil
}

func (lu *Luma) SetHosts(tree *trie.DomainTrie[resolver.HostValue]) {
	lu.mu.Lock()
	defer lu.mu.Unlock()
	lu.hosts = tree
}

func (lu *Luma) SetProxies(proxies map[string]proxy.Proxy) {
	lu.mu.Lock()
	lu.proxies = proxies
	lu.mu.Unlock()
}

func (lu *Luma) SetProxyProviders(providers map[string]provider.ProxyProvider) {
	lu.mu.Lock()
	lu.providers = providers
	lu.mu.Unlock()
}

// parseProxies returns a map of proxies that are present in the config
func parseProxies(cfg *config.Config) (map[string]proxy.Proxy, map[string]provider.ProxyProvider, error) {
	proxies := make(map[string]proxy.Proxy)
	providersMap := make(map[string]provider.ProxyProvider)
	var proxyList []string
	var AllProxies []string
	proxiesList := list.New()

	proxies["DIRECT"] = adapter.NewProxy(outbound.NewDirect())
	proxies["REJECT"] = adapter.NewProxy(outbound.NewReject())
	proxies["REJECT-DROP"] = adapter.NewProxy(outbound.NewRejectDrop())
	proxies["COMPATIBLE"] = adapter.NewProxy(outbound.NewCompatible())
	proxies["PASS"] = adapter.NewProxy(outbound.NewPass())
	proxyList = append(proxyList, "DIRECT", "REJECT")
	// parse proxy
	for idx, mapping := range cfg.RawProxies {
		proxy, err := adapter.ParseProxy(mapping)
		if err != nil {
			return nil, nil, fmt.Errorf("proxy %d: %w", idx, err)
		}

		if _, exist := proxies[proxy.Name()]; exist {
			return nil, nil, fmt.Errorf("proxy %s is the duplicate name", proxy.Name())
		}
		log.Debugf("Adding proxy named %s", proxy.Name())
		proxies[proxy.Name()] = proxy
		proxyList = append(proxyList, proxy.Name())
		AllProxies = append(AllProxies, proxy.Name())
		proxiesList.PushBack(mapping)
	}

	// parse and initial providers
	var AllProviders []string
	for name, mapping := range cfg.ProxyProvider {
		if name == provider.ReservedName {
			return nil, nil, fmt.Errorf("can not defined a provider called `%s`", provider.ReservedName)
		}

		pd, err := provider.ParseProxyProvider(name, mapping)
		if err != nil {
			return nil, nil, fmt.Errorf("parse proxy provider %s error: %w", name, err)
		}

		providersMap[name] = pd
		AllProviders = append(AllProviders, name)
	}

	return proxies, providersMap, nil
}

func configureTun(cfg *config.Config, tunnel tunnel.Tunnel) error {
	rawTun := cfg.RawTun
	tunAddressPrefix := tunnel.FakeIPRange()
	if !tunAddressPrefix.IsValid() {
		tunAddressPrefix = netip.MustParsePrefix("198.18.0.1/16")
	}
	tunAddressPrefix = netip.PrefixFrom(tunAddressPrefix.Addr(), 30)

	if !cfg.IPv6 || !verifyIP6() {
		rawTun.Inet6Address = nil
	}

	cfg.Tun = &config.Tun{
		Enable:                   rawTun.Enable,
		Device:                   rawTun.Device,
		Stack:                    rawTun.Stack,
		DNSHijack:                rawTun.DNSHijack,
		AutoRoute:                rawTun.AutoRoute,
		AutoDetectInterface:      rawTun.AutoDetectInterface,
		DisableInterfaceMonitor:  rawTun.DisableInterfaceMonitor,
		BuildAndroidRules:        rawTun.BuildAndroidRules,
		RedirectToTun:            rawTun.RedirectToTun,
		MTU:                      rawTun.MTU,
		GSO:                      rawTun.GSO,
		GSOMaxSize:               rawTun.GSOMaxSize,
		Inet4Address:             []netip.Prefix{tunAddressPrefix},
		Inet6Address:             rawTun.Inet6Address,
		StrictRoute:              rawTun.StrictRoute,
		Inet4RouteAddress:        rawTun.Inet4RouteAddress,
		Inet6RouteAddress:        rawTun.Inet6RouteAddress,
		Inet4RouteExcludeAddress: rawTun.Inet4RouteExcludeAddress,
		Inet6RouteExcludeAddress: rawTun.Inet6RouteExcludeAddress,
		IncludeInterface:         rawTun.IncludeInterface,
		ExcludeInterface:         rawTun.ExcludeInterface,
		IncludeUID:               rawTun.IncludeUID,
		IncludeUIDRange:          rawTun.IncludeUIDRange,
		ExcludeUID:               rawTun.ExcludeUID,
		ExcludeUIDRange:          rawTun.ExcludeUIDRange,
		IncludeAndroidUser:       rawTun.IncludeAndroidUser,
		IncludePackage:           rawTun.IncludePackage,
		ExcludePackage:           rawTun.ExcludePackage,
		EndpointIndependentNat:   rawTun.EndpointIndependentNat,
		UDPTimeout:               rawTun.UDPTimeout,
		FileDescriptor:           rawTun.FileDescriptor,
		TableIndex:               rawTun.TableIndex,
	}
	return nil
}
