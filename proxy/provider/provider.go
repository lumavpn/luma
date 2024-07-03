package provider

import (
	"sync"

	C "github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/log"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/rule"
	"github.com/lumavpn/luma/util"
)

// Provider Type
const (
	Proxy ProviderType = iota
	Rule
)

// ProviderType defined
type ProviderType int

func (pt ProviderType) String() string {
	switch pt {
	case Proxy:
		return "Proxy"
	case Rule:
		return "Rule"
	default:
		return "Unknown"
	}
}

// Provider interface
type Provider interface {
	Name() string
	VehicleType() C.VehicleType
	Type() ProviderType
	Initial() error
	Update() error
}

// ProxyProvider interface
type ProxyProvider interface {
	Provider
	Proxies() []proxy.Proxy
	// Touch is used to inform the provider that the proxy is actually being used while getting the list of proxies.
	// Commonly used in DialContext and DialPacketConn
	Touch()
	HealthCheck()
	Version() uint32
	RegisterHealthCheckTask(url string, expectedStatus util.IntRanges[uint16], filter string, interval uint)
	HealthCheckURL() string
}

// RuleProvider interface
type RuleProvider interface {
	Provider
	Behavior() RuleBehavior
	Match(*M.Metadata) bool
	ShouldResolveIP() bool
	ShouldFindProcess() bool
	AsRule(adaptor string) rule.Rule
}

// Rule Behavior
const (
	Domain RuleBehavior = iota
	IPCIDR
	Classical
)

// RuleBehavior defined
type RuleBehavior int

func (rt RuleBehavior) String() string {
	switch rt {
	case Domain:
		return "Domain"
	case IPCIDR:
		return "IPCIDR"
	case Classical:
		return "Classical"
	default:
		return "Unknown"
	}
}

const (
	YamlRule RuleFormat = iota
	TextRule
)

type RuleFormat int

func (rf RuleFormat) String() string {
	switch rf {
	case YamlRule:
		return "YamlRule"
	case TextRule:
		return "TextRule"
	default:
		return "Unknown"
	}
}

func loadProvider(pv Provider) {
	if pv.VehicleType() == C.Compatible {
		return
	} else {
		log.Infof("Start initial provider %s", (pv).Name())
	}

	if err := pv.Initial(); err != nil {
		switch pv.Type() {
		case Proxy:
			{
				log.Errorf("initial proxy provider %s error: %v", (pv).Name(), err)
			}
		case Rule:
			{
				log.Errorf("initial rule provider %s error: %v", (pv).Name(), err)
			}
		}
	}
}

func HCCompatibleProvider(proxyProviders map[string]ProxyProvider) {
	// limit concurrent size
	wg := sync.WaitGroup{}
	ch := make(chan struct{}, concurrentCount)
	for _, proxyProvider := range proxyProviders {
		proxyProvider := proxyProvider
		if proxyProvider.VehicleType() == C.Compatible {
			log.Infof("Start initial Compatible provider %s", proxyProvider.Name())
			wg.Add(1)
			ch <- struct{}{}
			go func() {
				defer func() { <-ch; wg.Done() }()
				if err := proxyProvider.Initial(); err != nil {
					log.Errorf("initial Compatible provider %s error: %v", proxyProvider.Name(), err)
				}
			}()
		}

	}

}

func LoadRuleProvider(ruleProviders map[string]RuleProvider) {
	wg := sync.WaitGroup{}
	ch := make(chan struct{}, concurrentCount)
	for _, ruleProvider := range ruleProviders {
		ruleProvider := ruleProvider
		wg.Add(1)
		ch <- struct{}{}
		go func() {
			defer func() { <-ch; wg.Done() }()
			loadProvider(ruleProvider)

		}()
	}

	wg.Wait()
}

func LoadProxyProvider(proxyProviders map[string]ProxyProvider) {
	// limit concurrent size
	wg := sync.WaitGroup{}
	ch := make(chan struct{}, concurrentCount)
	for _, proxyProvider := range proxyProviders {
		proxyProvider := proxyProvider
		wg.Add(1)
		ch <- struct{}{}
		go func() {
			defer func() { <-ch; wg.Done() }()
			loadProvider(proxyProvider)
		}()
	}

	wg.Wait()
}
