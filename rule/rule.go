package rule

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/metadata"
)

type Rule interface {
	Match(*metadata.Metadata) (bool, string)
	Adapter() string
	Payload() string
	Rule() RuleType
	ShouldResolveIP() bool
	ShouldFindProcess() bool
}

func EncodeRuleType(s string) (RuleType, error) {
	r, ok := RuleType_value[strings.ToUpper(s)]
	if !ok {
		return RuleType_Unset, errors.New("Unknown rule")
	}

	return RuleType(r), nil
}

func ParseRule(rt, payload, target string, params []string) (Rule, error) {
	ruleType, err := EncodeRuleType(rt)
	if err != nil {
		return nil, err
	}
	var rule Rule
	switch ruleType {
	case RuleType_DOMAIN:
		rule = NewDomain(payload, target)
	case RuleType_NETWORK:
		rule, err = NewNetworkType(payload, target)
	default:
		return nil, fmt.Errorf("Unknown rule type: %v", rt)
	}
	if err != nil {
		log.Errorf("Unable to process %s rule: %v", rt, err)
		return nil, err
	}
	return rule, nil
}
