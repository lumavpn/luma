package luma

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/listener"
	"github.com/lumavpn/luma/listener/inbound"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/adapter"
	"github.com/lumavpn/luma/proxy/outbound"
	"github.com/lumavpn/luma/proxy/protos"
	"github.com/lumavpn/luma/rule"
	"github.com/lumavpn/luma/tunnel"
	"github.com/lumavpn/luma/util"
)

// parseConfig is used to parse the general configuration used by Luma
func (lu *Luma) parseConfig(cfg *config.Config) error {
	proxies, err := parseProxies(cfg)
	if err != nil {
		return err
	}
	listeners, err := parseListeners(cfg)
	if err != nil {
		return err
	}

	rules, err := parseRules(cfg, proxies, "rules")
	if err != nil {
		return err
	}

	log.Debugf("Have %d rules", len(rules))

	lu.mu.Lock()
	lu.listeners = listeners
	lu.proxies = proxies
	lu.mu.Unlock()
	return nil
}

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

func parseRules(cfg *config.Config, proxies map[string]proxy.Proxy, format string) ([]rule.Rule, error) {
	var rules []rule.Rule
	for idx, line := range cfg.Rules {
		ruleParts := util.TrimArray(strings.Split(line, ","))
		if len(ruleParts) == 0 {
			log.Errorf("Invalid rule line, skipping: %v", line)
			continue
		}
		var target, payload string
		var params []string
		ruleName := ruleParts[0]
		ruleLength := len(ruleParts)
		if ruleName == "NOT" || ruleName == "OR" || ruleName == "AND" || ruleName == "SUB-RULE" {
			target = ruleParts[ruleLength-1]
			payload = strings.Join(ruleParts[1:ruleLength-1], ",")
		} else {
			if ruleLength < 2 {
				return nil, fmt.Errorf("%s[%d] [%s] error: format invalid", format, idx, line)
			}
			if ruleLength < 4 {
				ruleParts = append(ruleParts, make([]string, 4-ruleLength)...)
			}
			if ruleName == "MATCH" {
				ruleLength = 2
			}
			if ruleLength >= 3 {
				ruleLength = 3
				payload = ruleParts[1]
			}
			target = ruleParts[ruleLength-1]
			params = ruleParts[ruleLength:]
		}
		params = util.TrimArray(params)
		rule, err := rule.ParseRule(ruleName, payload, target, params)
		if err != nil {
			log.Errorf("Unknown rule: %v", ruleName)
			continue
		}
		rules = append(rules, rule)
	}
	return rules, nil
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
