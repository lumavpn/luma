package proxydialer

import "github.com/lumavpn/luma/proxy"

// ProxyDialer is the primary mechanism for dialing proxies that Luma is configured to use
type ProxyDialer interface {
	AddProxies(map[string]proxy.Proxy) error
}

type proxyDialer struct {
	proxies map[string]proxy.Proxy
}

// New creates a new instance of ProxyDialer
func New() ProxyDialer {
	return &proxyDialer{
		proxies: make(map[string]proxy.Proxy),
	}
}

// AddProxies adds the proxies to include when dialing connections
func (pd *proxyDialer) AddProxies(proxies map[string]proxy.Proxy) error {
	return nil
}
