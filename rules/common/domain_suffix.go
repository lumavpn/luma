package common

import (
	"strings"

	"github.com/lumavpn/luma/log"
	M "github.com/lumavpn/luma/metadata"
	C "github.com/lumavpn/luma/rule"
	"golang.org/x/net/idna"
)

type DomainSuffix struct {
	*Base
	suffix  string
	adapter string
}

func (ds *DomainSuffix) RuleType() C.RuleType {
	return C.DomainSuffix
}

func (ds *DomainSuffix) Match(metadata *M.Metadata) (bool, string) {
	domain := metadata.RuleHost()
	log.Debugf("Calling DomainSuffix Match .. domain is %s suffix %s", domain, ds.suffix)
	return strings.HasSuffix(domain, "."+ds.suffix) || domain == ds.suffix, ds.adapter
}

func (ds *DomainSuffix) Adapter() string {
	return ds.adapter
}

func (ds *DomainSuffix) Payload() string {
	return ds.suffix
}

func NewDomainSuffix(suffix string, adapter string) *DomainSuffix {
	punycode, _ := idna.ToASCII(strings.ToLower(suffix))
	return &DomainSuffix{
		Base:    &Base{},
		suffix:  punycode,
		adapter: adapter,
	}
}

//var _ C.Rule = (*DomainSuffix)(nil)
