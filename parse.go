package luma

import (
	"fmt"
	"net/netip"

	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/local"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/adapter"
	"github.com/lumavpn/luma/proxy/outbound"
	"github.com/lumavpn/luma/proxy/proto"
)

type configResult struct {
	locals  map[string]local.LocalServer
	proxies map[string]proxy.Proxy
}

func (lu *Luma) SetProxies(proxies map[string]proxy.Proxy) {
	lu.mu.Lock()
	defer lu.mu.Unlock()
	lu.proxies = proxies
}

// parseConfig is used to parse the general configuration used by Luma
func (lu *Luma) parseConfig(cfg *config.Config) (map[string]proxy.Proxy, map[string]local.LocalServer, error) {
	proxies, err := parseProxies(cfg)
	if err != nil {
		return nil, nil, err
	}

	log.Debugf("Have %d proxies", len(proxies))

	localServers, err := parseLocal(cfg)
	if err != nil {
		return nil, nil, err
	}

	cfg.Tun, err = lu.parseTun(cfg)
	if err != nil {
		log.Fatalf("unable to parse tun config: %v", err)
	}

	log.Debugf("Have %d local servers", len(localServers))
	return proxies, localServers, nil
}

func (lu *Luma) parseTun(cfg *config.Config) (*config.Tun, error) {
	rawTun := cfg.Tun
	tunAddressPrefix := lu.tunnel.FakeIPRange()
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
		Inet4Address:             []netip.Prefix{tunAddressPrefix},
		Inet6Address:             rawTun.Inet6Address,
		Inet4RouteAddress:        rawTun.Inet4RouteAddress,
		Inet6RouteAddress:        rawTun.Inet6RouteAddress,
		Inet4RouteExcludeAddress: rawTun.Inet4RouteExcludeAddress,
		Inet6RouteExcludeAddress: rawTun.Inet6RouteExcludeAddress,
	}
	return tc, nil
}

// parseProxies returns a map of proxies that are present in the config
func parseProxies(cfg *config.Config) (map[string]proxy.Proxy, error) {
	proxies := make(map[string]proxy.Proxy)
	proxies[proto.Proto_DIRECT.String()] = adapter.NewProxy(outbound.NewDirect())
	for _, mapping := range cfg.Proxies {
		proxy, err := adapter.ParseProxy(mapping)
		if err != nil {
			return nil, fmt.Errorf("error parsing proxy %w", err)
		}

		if _, exist := proxies[proxy.Name()]; exist {
			return nil, fmt.Errorf("proxy %s is the duplicate name", proxy.Name())
		}
		log.Debugf("Adding proxy named %s", proxy.Name())
		proxies[proxy.Name()] = proxy
	}
	return proxies, nil
}

// parseLocal returns a map of local proxy servers that are currently running
func parseLocal(cfg *config.Config) (map[string]local.LocalServer, error) {
	servers := make(map[string]local.LocalServer)
	for index, mapping := range cfg.Locals {
		server, err := local.ParseLocal(mapping)
		if err != nil {
			return nil, fmt.Errorf("parse local server %d: %w", index, err)
		} else if _, exist := mapping[server.Name()]; exist {
			return nil, fmt.Errorf("server %s is the duplicate name", server.Name())
		}
		servers[server.Name()] = server
	}
	return servers, nil
}
