package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/dlclark/regexp2"
	C "github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/common/convert"
	lumaHttp "github.com/lumavpn/luma/component/http"
	"github.com/lumavpn/luma/component/resource"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/adapter"
	"github.com/lumavpn/luma/tunnel/statistic"
	"github.com/lumavpn/luma/util"
	"gopkg.in/yaml.v3"
)

const (
	ReservedName = "default"
)

type ProxySchema struct {
	Proxies []map[string]any `yaml:"proxies"`
}

// ProxySetProvider for auto gc
type ProxySetProvider struct {
	*proxySetProvider
}

type proxySetProvider struct {
	*resource.Fetcher[[]proxy.Proxy]
	proxies          []proxy.Proxy
	healthCheck      *HealthCheck
	version          uint32
	subscriptionInfo *SubscriptionInfo
}

func (pp *proxySetProvider) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"name":             pp.Name(),
		"type":             pp.Type().String(),
		"vehicleType":      pp.VehicleType().String(),
		"proxies":          pp.Proxies(),
		"testUrl":          pp.healthCheck.url,
		"expectedStatus":   pp.healthCheck.expectedStatus.String(),
		"updatedAt":        pp.UpdatedAt,
		"subscriptionInfo": pp.subscriptionInfo,
	})
}

func (pp *proxySetProvider) Version() uint32 {
	return pp.version
}

func (pp *proxySetProvider) Name() string {
	return pp.Fetcher.Name()
}

func (pp *proxySetProvider) HealthCheck() {
	pp.healthCheck.check()
}

func (pp *proxySetProvider) Update() error {
	elm, same, err := pp.Fetcher.Update()
	if err == nil && !same {
		pp.OnUpdate(elm)
	}
	return err
}

func (pp *proxySetProvider) Initial() error {
	elm, err := pp.Fetcher.Initial()
	if err != nil {
		return err
	}
	pp.OnUpdate(elm)
	pp.getSubscriptionInfo()
	pp.closeAllConnections()
	return nil
}

func (pp *proxySetProvider) Type() ProviderType {
	return Proxy
}

func (pp *proxySetProvider) Proxies() []proxy.Proxy {
	return pp.proxies
}

func (pp *proxySetProvider) Touch() {
	pp.healthCheck.touch()
}

func (pp *proxySetProvider) HealthCheckURL() string {
	return pp.healthCheck.url
}

func (pp *proxySetProvider) RegisterHealthCheckTask(url string, expectedStatus util.IntRanges[uint16], filter string, interval uint) {
	pp.healthCheck.registerHealthCheckTask(url, expectedStatus, filter, interval)
}

func (pp *proxySetProvider) setProxies(proxies []proxy.Proxy) {
	pp.proxies = proxies
	pp.healthCheck.setProxy(proxies)
	if pp.healthCheck.auto() {
		go pp.healthCheck.check()
	}
}

func (pp *proxySetProvider) getSubscriptionInfo() {
	if pp.VehicleType() != C.HTTP {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*90)
		defer cancel()
		resp, err := lumaHttp.HttpRequestWithProxy(ctx, pp.Vehicle().(*resource.HTTPVehicle).Url(),
			http.MethodGet, http.Header{"User-Agent": {C.UA}}, nil, pp.Vehicle().Proxy())
		if err != nil {
			return
		}
		defer resp.Body.Close()

		userInfoStr := strings.TrimSpace(resp.Header.Get("subscription-userinfo"))
		if userInfoStr == "" {
			resp2, err := lumaHttp.HttpRequestWithProxy(ctx, pp.Vehicle().(*resource.HTTPVehicle).Url(),
				http.MethodGet, http.Header{"User-Agent": {"Quantumultx"}}, nil, pp.Vehicle().Proxy())
			if err != nil {
				return
			}
			defer resp2.Body.Close()
			userInfoStr = strings.TrimSpace(resp2.Header.Get("subscription-userinfo"))
			if userInfoStr == "" {
				return
			}
		}
		pp.subscriptionInfo, err = NewSubscriptionInfo(userInfoStr)
		if err != nil {
			log.Warnf("[Provider] get subscription-userinfo: %e", err)
		}
	}()
}

func (pp *proxySetProvider) closeAllConnections() {
	statistic.DefaultManager.Range(func(c statistic.Tracker) bool {
		for _, chain := range c.Chains() {
			if chain == pp.Name() {
				_ = c.Close()
				break
			}
		}
		return true
	})
}

func stopProxyProvider(pd *ProxySetProvider) {
	pd.healthCheck.close()
	_ = pd.Fetcher.Destroy()
}

func NewProxySetProvider(name string, interval time.Duration, filter string, excludeFilter string, excludeType string,
	dialerProxy string, override OverrideSchema, vehicle C.Vehicle, hc *HealthCheck) (*ProxySetProvider, error) {
	excludeFilterReg, err := regexp2.Compile(excludeFilter, 0)
	if err != nil {
		return nil, fmt.Errorf("invalid excludeFilter regex: %w", err)
	}
	var excludeTypeArray []string
	if excludeType != "" {
		excludeTypeArray = strings.Split(excludeType, "|")
	}

	var filterRegs []*regexp2.Regexp
	for _, filter := range strings.Split(filter, "`") {
		filterReg, err := regexp2.Compile(filter, 0)
		if err != nil {
			return nil, fmt.Errorf("invalid filter regex: %w", err)
		}
		filterRegs = append(filterRegs, filterReg)
	}

	if hc.auto() {
		go hc.process()
	}

	pd := &proxySetProvider{
		proxies:     []proxy.Proxy{},
		healthCheck: hc,
	}

	fetcher := resource.NewFetcher[[]proxy.Proxy](name, interval, vehicle, proxiesParseAndFilter(filter, excludeFilter, excludeTypeArray, filterRegs, excludeFilterReg, dialerProxy, override), proxiesOnUpdate(pd))
	pd.Fetcher = fetcher
	wrapper := &ProxySetProvider{pd}
	runtime.SetFinalizer(wrapper, stopProxyProvider)
	return wrapper, nil
}

// CompatibleProvider for auto gc
type CompatibleProvider struct {
	*compatibleProvider
}

type compatibleProvider struct {
	name        string
	healthCheck *HealthCheck
	proxies     []proxy.Proxy
	version     uint32
}

func (cp *compatibleProvider) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"name":           cp.Name(),
		"type":           cp.Type().String(),
		"vehicleType":    cp.VehicleType().String(),
		"proxies":        cp.Proxies(),
		"testUrl":        cp.healthCheck.url,
		"expectedStatus": cp.healthCheck.expectedStatus.String(),
	})
}

func (cp *compatibleProvider) Version() uint32 {
	return cp.version
}

func (cp *compatibleProvider) Name() string {
	return cp.name
}

func (cp *compatibleProvider) HealthCheck() {
	cp.healthCheck.check()
}

func (cp *compatibleProvider) Update() error {
	return nil
}

func (cp *compatibleProvider) Initial() error {
	if cp.healthCheck.interval != 0 && cp.healthCheck.url != "" {
		cp.HealthCheck()
	}
	return nil
}

func (cp *compatibleProvider) VehicleType() C.VehicleType {
	return C.Compatible
}

func (cp *compatibleProvider) Type() ProviderType {
	return Proxy
}

func (cp *compatibleProvider) Proxies() []proxy.Proxy {
	return cp.proxies
}

func (cp *compatibleProvider) Touch() {
	cp.healthCheck.touch()
}

func (cp *compatibleProvider) HealthCheckURL() string {
	return cp.healthCheck.url
}

func (cp *compatibleProvider) RegisterHealthCheckTask(url string, expectedStatus util.IntRanges[uint16], filter string, interval uint) {
	cp.healthCheck.registerHealthCheckTask(url, expectedStatus, filter, interval)
}

func stopCompatibleProvider(pd *CompatibleProvider) {
	pd.healthCheck.close()
}

func NewCompatibleProvider(name string, proxies []proxy.Proxy, hc *HealthCheck) (*CompatibleProvider, error) {
	if len(proxies) == 0 {
		return nil, errors.New("provider need one proxy at least")
	}

	if hc.auto() {
		go hc.process()
	}

	pd := &compatibleProvider{
		name:        name,
		proxies:     proxies,
		healthCheck: hc,
	}

	wrapper := &CompatibleProvider{pd}
	runtime.SetFinalizer(wrapper, stopCompatibleProvider)
	return wrapper, nil
}

func proxiesOnUpdate(pd *proxySetProvider) func([]proxy.Proxy) {
	return func(elm []proxy.Proxy) {
		pd.setProxies(elm)
		pd.version += 1
		pd.getSubscriptionInfo()
	}
}

func proxiesParseAndFilter(filter string, excludeFilter string, excludeTypeArray []string, filterRegs []*regexp2.Regexp, excludeFilterReg *regexp2.Regexp, dialerProxy string, override OverrideSchema) resource.Parser[[]proxy.Proxy] {
	return func(buf []byte) ([]proxy.Proxy, error) {
		schema := &ProxySchema{}

		if err := yaml.Unmarshal(buf, schema); err != nil {
			proxies, err1 := convert.ConvertsV2Ray(buf)
			if err1 != nil {
				return nil, fmt.Errorf("%w, %w", err, err1)
			}
			schema.Proxies = proxies
		}

		if schema.Proxies == nil {
			return nil, errors.New("file must have a `proxies` field")
		}

		proxies := []proxy.Proxy{}
		proxiesSet := map[string]struct{}{}
		for _, filterReg := range filterRegs {
			for idx, mapping := range schema.Proxies {
				if nil != excludeTypeArray && len(excludeTypeArray) > 0 {
					mType, ok := mapping["type"]
					if !ok {
						continue
					}
					pType, ok := mType.(string)
					if !ok {
						continue
					}
					flag := false
					for i := range excludeTypeArray {
						if strings.EqualFold(pType, excludeTypeArray[i]) {
							flag = true
							break
						}

					}
					if flag {
						continue
					}

				}
				mName, ok := mapping["name"]
				if !ok {
					continue
				}
				name, ok := mName.(string)
				if !ok {
					continue
				}
				if len(excludeFilter) > 0 {
					if mat, _ := excludeFilterReg.FindStringMatch(name); mat != nil {
						continue
					}
				}
				if len(filter) > 0 {
					if mat, _ := filterReg.FindStringMatch(name); mat == nil {
						continue
					}
				}
				if _, ok := proxiesSet[name]; ok {
					continue
				}

				if len(dialerProxy) > 0 {
					mapping["dialer-proxy"] = dialerProxy
				}

				if override.UDP != nil {
					mapping["udp"] = *override.UDP
				}
				if override.Up != nil {
					mapping["up"] = *override.Up
				}
				if override.Down != nil {
					mapping["down"] = *override.Down
				}
				if override.DialerProxy != nil {
					mapping["dialer-proxy"] = *override.DialerProxy
				}
				if override.SkipCertVerify != nil {
					mapping["skip-cert-verify"] = *override.SkipCertVerify
				}
				if override.Interface != nil {
					mapping["interface-name"] = *override.Interface
				}
				if override.RoutingMark != nil {
					mapping["routing-mark"] = *override.RoutingMark
				}
				if override.IPVersion != nil {
					mapping["ip-version"] = *override.IPVersion
				}
				if override.AdditionalPrefix != nil {
					name := mapping["name"].(string)
					mapping["name"] = *override.AdditionalPrefix + name
				}
				if override.AdditionalSuffix != nil {
					name := mapping["name"].(string)
					mapping["name"] = name + *override.AdditionalSuffix
				}

				proxy, err := adapter.ParseProxy(mapping)
				if err != nil {
					return nil, fmt.Errorf("proxy %d error: %w", idx, err)
				}

				proxiesSet[name] = struct{}{}
				proxies = append(proxies, proxy)
			}
		}

		if len(proxies) == 0 {
			if len(filter) > 0 {
				return nil, errors.New("doesn't match any proxy, please check your filter")
			}
			return nil, errors.New("file doesn't have any proxy")
		}

		return proxies, nil
	}
}
