package rule

import (
	"strings"

	"github.com/lumavpn/luma/metadata"
	"golang.org/x/net/idna"
)

type Domain struct {
	*Base
	domain  string
	adapter string
}

func NewDomain(domain string, adapter string) *Domain {
	punycode, _ := idna.ToASCII(strings.ToLower(domain))
	return &Domain{
		Base:    NewBase(RuleType_DOMAIN, adapter),
		domain:  punycode,
		adapter: adapter,
	}
}

func (d *Domain) Match(m *metadata.Metadata) (bool, string) {
	return m.Host == d.domain, d.adapter
}

func (d *Domain) Payload() string {
	return d.domain
}
