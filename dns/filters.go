package dns

import "net/netip"

type fallbackIPFilter interface {
	Match(netip.Addr) bool
}

type fallbackDomainFilter interface {
	Match(domain string) bool
}
