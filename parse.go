package luma

import (
	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
)

// parseConfig is used to parse the general configuration used by Luma
func (lu *Luma) parseConfig(cfg *config.Config) error {
	proxies, err := parseProxies(cfg)
	if err != nil {
		return err
	}

	log.Debugf("Have %d proxies", len(proxies))

	lu.mu.Lock()
	lu.proxies = proxies
	lu.mu.Unlock()
	return nil
}

// parseProxies returns a map of proxies that are present in the config
func parseProxies(cfg *config.Config) (map[string]proxy.Proxy, error) {
	proxies := make(map[string]proxy.Proxy)
	return proxies, nil
}
