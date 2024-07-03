package luma

import (
	"fmt"
	"strings"

	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/provider"
	"github.com/lumavpn/luma/rule"
	R "github.com/lumavpn/luma/rule"
	RR "github.com/lumavpn/luma/rules"
	RP "github.com/lumavpn/luma/rules/provider"
)

func trimArr(arr []string) (r []string) {
	for _, e := range arr {
		r = append(r, strings.Trim(e, " "))
	}
	return
}

func (lu *Luma) SetRuleProviders(ruleProviders map[string]provider.RuleProvider) {
	lu.mu.Lock()
	defer lu.mu.Unlock()
	lu.ruleProviders = ruleProviders
}

func (lu *Luma) SetRules(rules []rule.Rule) {
	lu.mu.Lock()
	defer lu.mu.Unlock()
	lu.rules = rules
}

func (lu *Luma) SetSubRules(subRules map[string][]rule.Rule) {
	lu.mu.Lock()
	defer lu.mu.Unlock()
	lu.subRules = subRules
}

func parseRules(rulesConfig []string, proxies map[string]proxy.Proxy, subRules map[string][]R.Rule, format string) ([]R.Rule, error) {
	var rules []R.Rule

	// parse rules
	for idx, line := range rulesConfig {
		rule := trimArr(strings.Split(line, ","))
		var (
			payload  string
			target   string
			params   []string
			ruleName = strings.ToUpper(rule[0])
		)

		l := len(rule)

		if ruleName == "NOT" || ruleName == "OR" || ruleName == "AND" || ruleName == "SUB-RULE" || ruleName == "DOMAIN-REGEX" {
			target = rule[l-1]
			payload = strings.Join(rule[1:l-1], ",")
		} else {
			if l < 2 {
				return nil, fmt.Errorf("%s[%d] [%s] error: format invalid", format, idx, line)
			}
			if l < 4 {
				rule = append(rule, make([]string, 4-l)...)
			}
			if ruleName == "MATCH" {
				l = 2
			}
			if l >= 3 {
				l = 3
				payload = rule[1]
			}
			target = rule[l-1]
			params = rule[l:]
		}
		if _, ok := proxies[target]; !ok {
			if ruleName != "SUB-RULE" {
				return nil, fmt.Errorf("%s[%d] [%s] error: proxy [%s] not found", format, idx, line, target)
			} else if _, ok = subRules[target]; !ok {
				return nil, fmt.Errorf("%s[%d] [%s] error: sub-rule [%s] not found", format, idx, line, target)
			}
		}

		params = trimArr(params)
		parsed, parseErr := RR.ParseRule(ruleName, payload, target, params, subRules)
		if parseErr != nil {
			return nil, fmt.Errorf("%s[%d] [%s] error: %s", format, idx, line, parseErr.Error())
		}

		rules = append(rules, parsed)
	}

	return rules, nil
}

func parseRuleProviders(providers map[string]map[string]any) (ruleProviders map[string]provider.RuleProvider, err error) {
	ruleProviders = map[string]provider.RuleProvider{}
	// parse rule provider
	for name, mapping := range providers {
		rp, err := RP.ParseRuleProvider(name, mapping, RR.ParseRule)
		if err != nil {
			return nil, err
		}

		ruleProviders[name] = rp
		RP.SetRuleProvider(rp)
	}
	return
}

func parseSubRules(rawSubRules map[string][]string, proxies map[string]proxy.Proxy) (subRules map[string][]R.Rule, err error) {
	subRules = map[string][]R.Rule{}
	for name := range rawSubRules {
		subRules[name] = make([]R.Rule, 0)
	}
	for name, rawRules := range rawSubRules {
		if len(name) == 0 {
			return nil, fmt.Errorf("sub-rule name is empty")
		}
		var rules []R.Rule
		rules, err = parseRules(rawRules, proxies, subRules, fmt.Sprintf("sub-rules[%s]", name))
		if err != nil {
			return nil, err
		}
		subRules[name] = rules
	}

	if err = verifySubRule(subRules); err != nil {
		return nil, err
	}

	return
}

func verifySubRule(subRules map[string][]R.Rule) error {
	for name := range subRules {
		err := verifySubRuleCircularReferences(name, subRules, []string{})
		if err != nil {
			return err
		}
	}
	return nil
}

func verifySubRuleCircularReferences(n string, subRules map[string][]R.Rule, arr []string) error {
	isInArray := func(v string, array []string) bool {
		for _, c := range array {
			if v == c {
				return true
			}
		}
		return false
	}

	arr = append(arr, n)
	for i, rule := range subRules[n] {
		if rule.RuleType() == R.SubRules {
			if _, ok := subRules[rule.Adapter()]; !ok {
				return fmt.Errorf("sub-rule[%d:%s] error: [%s] not found", i, n, rule.Adapter())
			}
			if isInArray(rule.Adapter(), arr) {
				arr = append(arr, rule.Adapter())
				return fmt.Errorf("sub-rule error: circular references [%s]", strings.Join(arr, "->"))
			}

			if err := verifySubRuleCircularReferences(rule.Adapter(), subRules, arr); err != nil {
				return err
			}
		}
	}
	return nil
}
