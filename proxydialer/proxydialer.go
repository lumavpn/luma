package proxydialer

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"

	"github.com/lumavpn/luma/component/resolver"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/protos"
	"github.com/lumavpn/luma/rule"
)

type proxyDialer struct {
	proxies  map[string]proxy.Proxy
	rules    []rule.Rule
	subRules map[string][]rule.Rule
	mu       *sync.RWMutex
}

type ProxyDialer interface {
	Match(m *metadata.Metadata) (proxy.Proxy, rule.Rule, error)
	SelectProxy(*metadata.Metadata) (proxy.Proxy, error)
	SelectProxyByName(string) (proxy.Proxy, error)
	UpdateProxies(map[string]proxy.Proxy)
	UpdateRules([]rule.Rule)
}

func New(proxies map[string]proxy.Proxy, rules []rule.Rule) ProxyDialer {
	return &proxyDialer{
		mu: new(sync.RWMutex),
	}
}

func (pd *proxyDialer) getRules(m *metadata.Metadata) []rule.Rule {
	if sr, ok := pd.subRules[m.SpecialRules]; ok {
		log.Debugf("[Rule] use %s rules", m.SpecialRules)
		return sr
	} else {
		log.Debug("[Rule] use default rules")
		return pd.rules
	}
}

func shouldResolveIP(rule rule.Rule, m *metadata.Metadata) bool {
	return rule.ShouldResolveIP() && m.Host != "" && !m.DstIP.IsValid()
}

func (pd *proxyDialer) Match(m *metadata.Metadata) (proxy.Proxy, rule.Rule, error) {
	pd.mu.RLock()
	defer pd.mu.RUnlock()
	var (
		resolved bool
	)

	if node, ok := resolver.DefaultHosts.Search(m.Host, false); ok {
		m.DstIP, _ = node.RandIP()
		resolved = true
	}
	for _, rule := range pd.getRules(m) {
		if !resolved && shouldResolveIP(rule, m) {
			ctx, cancel := context.WithTimeout(context.Background(), resolver.DefaultDNSTimeout)
			defer cancel()
			ip, err := resolver.ResolveIP(ctx, m.Host)
			if err != nil {
				log.Debugf("[DNS] resolve %s error: %s", m.Host, err.Error())
			} else {
				log.Debugf("[DNS] %s --> %s", m.Host, ip.String())
				m.DstIP = ip
			}
			resolved = true
		}

		if matched, ada := rule.Match(m); matched {
			adapter, ok := pd.proxies[ada]
			if !ok {
				continue
			}
			passed := false
			for adapter := adapter; adapter != nil; adapter = adapter.Unwrap(m, false) {
				if adapter.AdapterType() == protos.AdapterType_Pass {
					passed = true
					break
				}
			}
			if passed {
				log.Debugf("%s match Pass rule", adapter.Name())
				continue
			}

			if m.Network == metadata.UDP && !adapter.SupportUDP() {
				log.Debugf("%s UDP is not supported", adapter.Name())
				continue
			}
		}

	}

	return pd.proxies["DIRECT"], nil, nil
}

func (pd *proxyDialer) SelectProxy(m *metadata.Metadata) (proxy.Proxy, error) {
	pd.mu.Lock()
	defer pd.mu.RUnlock()
	proxiesMap := pd.proxies

	var proxies []proxy.Proxy
	for _, proxy := range proxiesMap {
		if m.Network == metadata.UDP && !proxy.SupportUDP() {
			continue
		}
		// filter direct
		if proxy.AdapterType() == protos.AdapterType_Direct ||
			proxy.AdapterType() == protos.AdapterType_Reject {
			continue
		}
		// filter global
		if proxy.Name() == "GLOBAL" {
			continue
		}
		proxies = append(proxies, proxy)
	}
	if len(proxies) == 0 {
		return nil, errors.New("No proxies available")
	}
	return proxies[rand.Intn(len(proxies))], nil
}

func (pd *proxyDialer) SelectProxyByName(name string) (proxy.Proxy, error) {
	pd.mu.RLock()
	proxies := pd.proxies
	pd.mu.RUnlock()
	if proxy, ok := proxies[name]; ok {
		return proxy, nil
	}
	return nil, fmt.Errorf("proxy %s not found", name)
}

// UpdateProxies handle update proxies
func (pd *proxyDialer) UpdateProxies(newProxies map[string]proxy.Proxy) {
	if len(newProxies) == 0 {
		return
	}
	log.Debugf("Re-configuring dialer with %d new proxies", len(newProxies))
	pd.mu.Lock()
	pd.proxies = newProxies
	pd.mu.Unlock()
}

// UpdateRules handle update rules
func (pd *proxyDialer) UpdateRules(newRules []rule.Rule) {
	pd.mu.Lock()
	pd.rules = newRules
	pd.mu.Unlock()
}
