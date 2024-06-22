package rule

import "github.com/lumavpn/luma/metadata"

type Base struct {
	rule    RuleType
	adapter string
}

func NewBase(r RuleType, adapter string) *Base {
	return &Base{
		rule:    r,
		adapter: adapter,
	}
}

func (b *Base) Adapter() string {
	return b.adapter
}

func (b *Base) Rule() RuleType {
	return b.rule
}

func (b *Base) Match(*metadata.Metadata) (bool, string) {
	return false, ""
}
