package luma

import (
	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/outbound"
	"github.com/lumavpn/luma/proxy/protos"
)

// parseProxies returns a map of proxies that are present in the config
func parseProxies(cfg *config.Config) (map[string]proxy.Proxy, error) {
	proxies := make(map[string]proxy.Proxy)
	proxies[protos.AdapterType_Direct.String()] = outbound.NewDirect()
	return proxies, nil
}
