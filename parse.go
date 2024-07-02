package luma

import (
	"net/netip"

	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/tunnel"
)

// parseConfig is used to parse and load the general configuration used by Luma
func (lu *Luma) parseConfig(cfg *config.Config) error {
	proxies, err := parseProxies(cfg)
	if err != nil {
		return err
	}
	lu.SetProxies(proxies)
	log.Debugf("Have %d proxies", len(proxies))

	if err := configureTun(cfg, lu.tunnel); err != nil {
		log.Fatalf("unable to parse tun config: %v", err)
	}

	return nil
}

func (lu *Luma) SetProxies(proxies map[string]proxy.Proxy) {
	lu.mu.Lock()
	lu.proxies = proxies
	lu.mu.Unlock()
}

// parseProxies returns a map of proxies that are present in the config
func parseProxies(cfg *config.Config) (map[string]proxy.Proxy, error) {
	proxies := make(map[string]proxy.Proxy)
	return proxies, nil
}

func configureTun(cfg *config.Config, tunnel tunnel.Tunnel) error {
	tunAddressPrefix := tunnel.FakeIPRange()
	if !tunAddressPrefix.IsValid() {
		tunAddressPrefix = netip.MustParsePrefix("198.18.0.1/16")
	}
	tunAddressPrefix = netip.PrefixFrom(tunAddressPrefix.Addr(), 30)
	cfg.Tun.Inet4Address = []netip.Prefix{tunAddressPrefix}

	if !cfg.IPv6 || !verifyIP6() {
		cfg.Tun.Inet6Address = nil
	}

	return nil
}
