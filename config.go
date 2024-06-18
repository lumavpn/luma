package luma

import (
	"fmt"
	"net/netip"

	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/listener"
	"github.com/lumavpn/luma/listener/inbound"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/adapter"
	"github.com/lumavpn/luma/proxy/outbound"
	"github.com/lumavpn/luma/proxy/protos"
	"github.com/lumavpn/luma/tunnel"
)

// parseProxies returns a map of proxies that are present in the config
func parseProxies(cfg *config.Config) (map[string]proxy.Proxy, error) {
	proxies := make(map[string]proxy.Proxy)
	proxies[protos.AdapterType_Direct.String()] = outbound.NewDirect()

	// parse proxy
	for idx, mapping := range cfg.Proxies {
		proxy, err := adapter.ParseProxy(mapping)
		if err != nil {
			return nil, fmt.Errorf("proxy %d: %w", idx, err)
		}

		if _, exist := proxies[proxy.Name()]; exist {
			return nil, fmt.Errorf("proxy %s is the duplicate name", proxy.Name())
		}
		log.Debugf("Adding proxy named %s", proxy.Name())
		proxies[proxy.Name()] = proxy
	}

	return proxies, nil
}

// parseListeners returns a map of listeners this instance of Luma is configured with
func parseListeners(cfg *config.Config) (listeners map[string]inbound.InboundListener, err error) {
	listeners = make(map[string]inbound.InboundListener)
	for index, mapping := range cfg.Listeners {
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

func parseTun(cfg *config.Config, t tunnel.Tunnel) (*config.Tun, error) {
	rawTun := cfg.Tun
	tunAddressPrefix := t.FakeIPRange()
	if !tunAddressPrefix.IsValid() {
		tunAddressPrefix = netip.MustParsePrefix("198.18.0.1/16")
	}
	tunAddressPrefix = netip.PrefixFrom(tunAddressPrefix.Addr(), 30)

	if !cfg.IPv6 || !verifyIP6() {
		rawTun.Inet6Address = nil
	}

	tc := &config.Tun{
		Enable:              rawTun.Enable,
		Device:              rawTun.Device,
		Stack:               rawTun.Stack,
		DNSHijack:           rawTun.DNSHijack,
		AutoRoute:           rawTun.AutoRoute,
		AutoDetectInterface: rawTun.AutoDetectInterface,
	}
	return tc, nil
}
