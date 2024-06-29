package provider

import (
	"sync"

	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
)

// ProxyProvider represents a proxy provider that Luma connects to provide proxies to users
type ProxyProvider interface {
	Provider
	Proxies() []proxy.Proxy
	Touch()
	HealthCheck()
	HealthCheckURL() string
}

func startProvider(pv Provider) {
	if err := pv.Initial(); err != nil {
		log.Errorf("initial %v provider %s error: %v", pv.Type(), pv.Name(), err)
	}
}

func LoadProxyProviders(proxyProviders map[string]ProxyProvider) {
	wg := sync.WaitGroup{}
	ch := make(chan struct{}, concurrentCount)
	for _, proxyProvider := range proxyProviders {
		proxyProvider := proxyProvider
		wg.Add(1)
		ch <- struct{}{}
		go func() {
			defer func() { <-ch; wg.Done() }()
			startProvider(proxyProvider)
		}()
	}

	wg.Wait()
}
