package luma

import (
	"container/list"
	"fmt"
	"net"
	"net/netip"

	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/listener"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/adapter"
	"github.com/lumavpn/luma/proxy/outbound"
	"github.com/lumavpn/luma/proxy/outboundgroup"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/tunnel"

	IN "github.com/lumavpn/luma/listener/inbound"
	"github.com/lumavpn/luma/proxy/provider"
	"github.com/lumavpn/luma/proxydialer"
)

var (
	ParsingProxiesCallback func(groupsList *list.List, proxiesList *list.List)
)

func parseListeners(cfg *config.Config) (listeners map[string]IN.InboundListener, err error) {
	listeners = make(map[string]IN.InboundListener)
	for index, mapping := range cfg.RawListeners {
		listener, err := listener.ParseListener(mapping)
		if err != nil {
			return nil, fmt.Errorf("proxy %d: %w", index, err)
		}

		if _, exist := mapping[listener.Name()]; exist {
			return nil, fmt.Errorf("listener %s is the duplicate name", listener.Name())
		}

		listeners[listener.Name()] = listener

	}
	return
}

func verifyIP6() bool {
	if iAddrs, err := net.InterfaceAddrs(); err == nil {
		for _, addr := range iAddrs {
			if prefix, err := netip.ParsePrefix(addr.String()); err == nil {
				if addr := prefix.Addr().Unmap(); addr.Is6() && addr.IsGlobalUnicast() {
					return true
				}
			}
		}
	}
	return false
}

func parseTun(cfg *config.Config, tunnel tunnel.Tunnel) (*config.Tun, error) {
	rawTun := cfg.RawTun
	tunAddressPrefix := tunnel.FakeIPRange()
	if !tunAddressPrefix.IsValid() {
		tunAddressPrefix = netip.MustParsePrefix("198.18.0.1/16")
	}
	tunAddressPrefix = netip.PrefixFrom(tunAddressPrefix.Addr(), 30)

	if !cfg.IPv6 || !verifyIP6() {
		rawTun.Inet6Address = nil
	}

	tc := &config.Tun{
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
	return tc, nil
}

func parseProxies(cfg *config.Config, proxyDialer proxydialer.ProxyDialer) (map[string]proxy.Proxy, map[string]provider.ProxyProvider, error) {
	proxies := make(map[string]proxy.Proxy)
	providersMap := make(map[string]provider.ProxyProvider)
	var proxyList []string
	var AllProxies []string
	proxiesList := list.New()
	groupsList := list.New()
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
	// keep the original order of ProxyGroups in config file
	for idx, mapping := range cfg.ProxyGroup {
		groupName, existName := mapping["name"].(string)
		if !existName {
			return nil, nil, fmt.Errorf("proxy group %d: missing name", idx)
		}
		proxyList = append(proxyList, groupName)
		groupsList.PushBack(mapping)
	}
	// check if any loop exists and sort the ProxyGroups
	if err := config.ProxyGroupsDagSort(cfg.ProxyGroup); err != nil {
		return nil, nil, err
	}
	var AllProviders []string
	// parse and initial providers
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

	// parse proxy group
	for idx, mapping := range cfg.ProxyGroup {
		group, err := outboundgroup.ParseProxyGroup(mapping, proxyDialer, proxies, providersMap, AllProxies, AllProviders)
		if err != nil {
			return nil, nil, fmt.Errorf("proxy group[%d]: %w", idx, err)
		}

		groupName := group.Name()
		if _, exist := proxies[groupName]; exist {
			return nil, nil, fmt.Errorf("proxy group %s: the duplicate name", groupName)
		}

		proxies[groupName] = adapter.NewProxy(group)
	}

	var ps []proxy.Proxy
	for _, v := range proxyList {
		if proxies[v].Proto() == proto.Proto_Pass {
			continue
		}
		ps = append(ps, proxies[v])
	}
	hc := provider.NewHealthCheck(ps, "", 5000, 0, true, nil)
	pd, _ := provider.NewCompatibleProvider(provider.ReservedName, ps, hc)
	providersMap[provider.ReservedName] = pd
	global := outboundgroup.NewSelector(
		&outboundgroup.GroupCommonOption{
			Name: "GLOBAL",
		},
		proxyDialer,
		[]provider.ProxyProvider{pd},
	)
	proxies["GLOBAL"] = adapter.NewProxy(global)

	if ParsingProxiesCallback != nil {
		// refresh tray menu
		go ParsingProxiesCallback(groupsList, proxiesList)
	}

	return proxies, providersMap, nil
}

func (lu *Luma) Proxies() map[string]proxy.Proxy {
	return lu.proxies
}

func (lu *Luma) SetListeners(listeners map[string]IN.InboundListener) {
	lu.mu.Lock()
	defer lu.mu.Unlock()
	lu.listeners = listeners
}

func (lu *Luma) SetProxies(proxies map[string]proxy.Proxy) {
	lu.mu.Lock()
	defer lu.mu.Unlock()
	lu.proxies = proxies
}

func (lu *Luma) SetProxyProviders(providers map[string]provider.ProxyProvider) {
	lu.mu.Lock()
	defer lu.mu.Unlock()
	lu.providers = providers
}
