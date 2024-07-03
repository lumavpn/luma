package proxydialer

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	C "github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/component/resolver"
	"github.com/lumavpn/luma/log"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/proxy/provider"
	"github.com/lumavpn/luma/rule"
)

var (
	errNoProxiesAvailable = errors.New("No proxies available")
)

type proxyDialer struct {
	mode          C.TunnelMode
	activeProxy   proxy.Proxy
	providers     map[string]provider.ProxyProvider
	proxies       map[string]proxy.Proxy
	rules         []rule.Rule
	ruleProviders map[string]provider.RuleProvider
	subRules      map[string][]rule.Rule
	mu            sync.RWMutex
}

type ProxyDialer interface {
	ActiveProtocols() []proto.Proto
	ActiveProxy() proxy.Proxy
	AddProxies(map[string]proxy.Proxy)
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
	ResolveMetadata(*M.Metadata) (proxy.Proxy, rule.Rule, error)
	Proxies() map[string]proxy.Proxy
	SetActiveProxy(string)
	SetPreferredProtocol(proto.Proto)
	SetMode(m C.TunnelMode)
	UpdateProxies(map[string]proxy.Proxy, map[string]provider.ProxyProvider)
	UpdateRules([]rule.Rule, map[string][]rule.Rule, map[string]provider.RuleProvider)
}

func New() ProxyDialer {
	return &proxyDialer{}
}

//var Dialer ProxyDialer = &proxyDialer{}

func newMetadata(network, address string) *M.Metadata {
	metadata := &M.Metadata{}
	n, _ := M.ParseNetwork(network)
	metadata.Network = n
	metadata.Type = proto.Proto_Inner
	metadata.DNSMode = C.DNSNormal
	if h, port, err := net.SplitHostPort(address); err == nil {
		if port, err := strconv.ParseUint(port, 10, 16); err == nil {
			metadata.DstPort = uint16(port)
		}
		if ip, err := netip.ParseAddr(h); err == nil {
			metadata.DstIP = ip
		} else {
			metadata.Host = h
		}
	}
	return metadata
}

func (pd *proxyDialer) ActiveProtocols() []proto.Proto {
	protoMap := make(map[proto.Proto]proxy.Proxy)
	pd.mu.Lock()
	proxies := pd.proxies
	pd.mu.Unlock()
	var protos []proto.Proto
	for _, proxy := range proxies {
		protoMap[proxy.Proto()] = proxy
	}
	for proto := range protoMap {
		protos = append(protos, proto)
	}

	return protos
}

func (pd *proxyDialer) ActiveProxy() proxy.Proxy {
	return pd.activeProxy
}

func (pd *proxyDialer) SetActiveProxy(name string) {
	pd.mu.Lock()
	proxies := pd.proxies
	pd.mu.Unlock()
	for proxyName, proxy := range proxies {
		if strings.EqualFold(proxyName, name) {
			pd.activeProxy = proxy
		}
	}
}

func (pd *proxyDialer) SetPreferredProtocol(proto proto.Proto) {

}

func (pd *proxyDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	metadata := newMetadata(network, addr)
	proxy, _, err := pd.ResolveMetadata(metadata)
	if err != nil {
		return nil, err
	} else if proxy == nil {
		return nil, errors.New("No proxy found")
	}
	return proxy.DialContext(ctx, metadata)
}

// UpdateProxies handle update proxies
func (pd *proxyDialer) UpdateProxies(newProxies map[string]proxy.Proxy, newProviders map[string]provider.ProxyProvider) {
	if len(newProxies) == 0 {
		return
	}
	log.Debugf("Re-configuring dialer with %d new proxies", len(newProxies))
	pd.mu.Lock()
	pd.proxies = newProxies
	pd.providers = newProviders
	pd.mu.Unlock()
}

func (pd *proxyDialer) AddProxies(newProxies map[string]proxy.Proxy) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	for _, proxy := range newProxies {
		if _, ok := pd.proxies[proxy.Name()]; ok {
			continue
		}
		pd.proxies[proxy.Name()] = proxy
	}
}

// SetMode change the mode of tunnel
func (pd *proxyDialer) SetMode(mo C.TunnelMode) {
	log.Debugf("Setting tunnel mode to %s", mo.String())
	pd.mu.Lock()
	pd.mode = mo
	pd.mu.Unlock()
}

// UpdateRules handle update rules
func (pd *proxyDialer) UpdateRules(newRules []rule.Rule, newSubRule map[string][]rule.Rule, rp map[string]provider.RuleProvider) {
	pd.mu.Lock()
	pd.rules = newRules
	pd.ruleProviders = rp
	pd.subRules = newSubRule
	pd.mu.Unlock()
}

// Providers return all compatible providers
func (pd *proxyDialer) Providers() map[string]provider.ProxyProvider {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	return pd.providers
}

// Proxies return all proxies
func (pd *proxyDialer) Proxies() map[string]proxy.Proxy {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	return pd.proxies
}

func (pd *proxyDialer) ResolveMetadata(metadata *M.Metadata) (proxy proxy.Proxy, rule rule.Rule, err error) {
	pd.mu.Lock()
	proxies := pd.proxies
	mode := pd.mode
	pd.mu.Unlock()
	if metadata.SpecialProxy != "" {
		var err error
		proxy, exist := proxies[metadata.SpecialProxy]
		if !exist {
			err = fmt.Errorf("proxy %s not found", metadata.SpecialProxy)
		}
		return proxy, nil, err
	}
	switch mode {
	// Select
	case C.Select:
		proxy, err = selectProxy(proxies, metadata)
	case C.Direct:
		proxy = proxies["DIRECT"]
	case C.Global:
		proxy = proxies["GLOBAL"]
	// Rule
	default:
		proxy, rule, err = pd.match(metadata)
	}
	return
}

func filterAdapterTypes(proxy proxy.Proxy) bool {
	return proxy.Proto() == proto.Proto_Direct || proxy.Proto() == proto.Proto_Compatible ||
		proxy.Proto() == proto.Proto_Reject || proxy.Proto() == proto.Proto_RejectDrop ||
		proxy.Proto() == proto.Proto_Pass
}

func selectProxy(proxiesMap map[string]proxy.Proxy, metadata *M.Metadata) (proxy.Proxy, error) {
	rand.Seed(time.Now().Unix())
	var proxies []proxy.Proxy
	for _, proxy := range proxiesMap {
		log.Debugf("Proxy name is %s", proxy.Name())
		if metadata.Network == M.UDP && !proxy.SupportUDP() {
			continue
		}
		// filter direct
		if filterAdapterTypes(proxy) {
			continue
		}
		// filter global
		if proxy.Name() == "GLOBAL" {
			continue
		}
		proxies = append(proxies, proxy)
	}
	if len(proxies) == 0 {
		return nil, errNoProxiesAvailable
	}
	proxy := proxies[rand.Intn(len(proxies))]
	log.Debugf("Selected proxy %s", proxy.Name())
	return proxy, nil
}

func (pd *proxyDialer) getRules(metadata *M.Metadata) []rule.Rule {
	if sr, ok := pd.subRules[metadata.SpecialRules]; ok {
		log.Debugf("[Rule] use %s rules", metadata.SpecialRules)
		return sr
	} else {
		log.Debug("[Rule] use default rules")
		return pd.rules
	}
}

func shouldResolveIP(rule rule.Rule, metadata *M.Metadata) bool {
	return rule.ShouldResolveIP() && metadata.Host != "" && !metadata.DstIP.IsValid()
}

func (pd *proxyDialer) match(metadata *M.Metadata) (proxy proxy.Proxy, rule rule.Rule, err error) {
	pd.mu.RLock()
	defer pd.mu.RUnlock()
	var (
		resolved bool
	)

	if node, ok := resolver.DefaultHosts.Search(metadata.Host, false); ok {
		metadata.DstIP, _ = node.RandIP()
		resolved = true
	}

	for _, rule := range pd.getRules(metadata) {
		if !resolved && shouldResolveIP(rule, metadata) {
			func() {
				ctx, cancel := context.WithTimeout(context.Background(), resolver.DefaultDNSTimeout)
				defer cancel()
				ip, err := resolver.ResolveIP(ctx, metadata.Host)
				if err != nil {
					log.Debugf("[DNS] resolve %s error: %s", metadata.Host, err.Error())
				} else {
					log.Debugf("[DNS] %s --> %s", metadata.Host, ip.String())
					metadata.DstIP = ip
				}
				resolved = true
			}()
		}

		if matched, ada := rule.Match(metadata); matched {
			adapter, ok := pd.proxies[ada]
			if !ok {
				continue
			}

			// parse multi-layer nesting
			passed := false
			for adapter := adapter; adapter != nil; adapter = adapter.Unwrap(metadata, false) {
				if adapter.Proto() == proto.Proto_Pass {
					passed = true
					break
				}
			}
			if passed {
				log.Debugf("%s match Pass rule", adapter.Name())
				continue
			}

			if metadata.Network == M.UDP && !adapter.SupportUDP() {
				log.Debugf("%s UDP is not supported", adapter.Name())
				continue
			}

			return adapter, rule, nil
		}

	}

	return pd.proxies["DIRECT"], nil, nil
}
