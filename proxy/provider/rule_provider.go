package provider

import (
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/rule"
)

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

// RuleProvider interface
type RuleProvider interface {
	Provider
	Behavior() RuleBehavior
	Match(*M.Metadata) bool
	ShouldResolveIP() bool
	ShouldFindProcess() bool
	AsRule(adaptor string) rule.Rule
}
