package dns

import (
	"github.com/lumavpn/luma/component/trie"
)

type dnsPolicy interface {
	Match(domain string) []dnsClient
}

type domainTriePolicy struct {
	*trie.DomainTrie[[]dnsClient]
}

func (p domainTriePolicy) Match(domain string) []dnsClient {
	record := p.DomainTrie.Search(domain)
	if record != nil {
		return record.Data()
	}
	return nil
}

type geositePolicy struct {
	matcher    fallbackDomainFilter
	inverse    bool
	dnsClients []dnsClient
}

func (p geositePolicy) Match(domain string) []dnsClient {
	matched := p.matcher.Match(domain)
	if matched != p.inverse {
		return p.dnsClients
	}
	return nil
}
