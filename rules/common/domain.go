package common

import (
	"strings"

	M "github.com/lumavpn/luma/metadata"
	C "github.com/lumavpn/luma/rule"
	"golang.org/x/net/idna"
)

type Domain struct {
	*Base
	domain  string
	adapter string
}

func (d *Domain) RuleType() C.RuleType {
	return C.Domain
}

func (d *Domain) Match(metadata *M.Metadata) (bool, string) {
	return metadata.RuleHost() == d.domain, d.adapter
}

func (d *Domain) Adapter() string {
	return d.adapter
}

func (d *Domain) Payload() string {
	return d.domain
}

func NewDomain(domain string, adapter string) *Domain {
	punycode, _ := idna.ToASCII(strings.ToLower(domain))
	return &Domain{
		Base:    &Base{},
		domain:  punycode,
		adapter: adapter,
	}
}

//var _ C.Rule = (*Domain)(nil)
