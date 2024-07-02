package config

import (
	"net/netip"

	C "github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/component/fakeip"
	"github.com/lumavpn/luma/component/trie"
	"github.com/lumavpn/luma/dns"
	"github.com/lumavpn/luma/dns/resolver"
	"github.com/lumavpn/luma/geodata/router"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// FallbackFilter config
type FallbackFilter struct {
	GeoIP     bool                   `yaml:"geoip"`
	GeoIPCode string                 `yaml:"geoip-code"`
	IPCIDR    []netip.Prefix         `yaml:"ipcidr"`
	Domain    []string               `yaml:"domain"`
	GeoSite   []router.DomainMatcher `yaml:"geosite"`
}

type RawDNS struct {
	Enable                bool                                `yaml:"enable" json:"enable"`
	PreferH3              bool                                `yaml:"prefer-h3" json:"prefer-h3"`
	IPv6                  bool                                `yaml:"ipv6" json:"ipv6"`
	IPv6Timeout           uint                                `yaml:"ipv6-timeout" json:"ipv6-timeout"`
	UseHosts              bool                                `yaml:"use-hosts" json:"use-hosts"`
	UseSystemHosts        bool                                `yaml:"use-system-hosts" json:"use-system-hosts"`
	NameServer            []string                            `yaml:"nameserver" json:"nameserver"`
	Fallback              []string                            `yaml:"fallback" json:"fallback"`
	FallbackFilter        RawFallbackFilter                   `yaml:"fallback-filter" json:"fallback-filter"`
	Listen                string                              `yaml:"listen" json:"listen"`
	EnhancedMode          C.DNSMode                           `yaml:"enhanced-mode" json:"enhanced-mode"`
	FakeIPRange           string                              `yaml:"fake-ip-range" json:"fake-ip-range"`
	FakeIPFilter          []string                            `yaml:"fake-ip-filter" json:"fake-ip-filter"`
	DefaultNameserver     []string                            `yaml:"default-nameserver" json:"default-nameserver"`
	CacheAlgorithm        string                              `yaml:"cache-algorithm" json:"cache-algorithm"`
	NameServerPolicy      *orderedmap.OrderedMap[string, any] `yaml:"nameserver-policy" json:"nameserver-policy"`
	ProxyServerNameserver []string                            `yaml:"proxy-server-nameserver" json:"proxy-server-nameserver"`
}

type RawFallbackFilter struct {
	GeoIP     bool     `yaml:"geoip" json:"geoip"`
	GeoIPCode string   `yaml:"geoip-code" json:"geoip-code"`
	IPCIDR    []string `yaml:"ipcidr" json:"ipcidr"`
	Domain    []string `yaml:"domain" json:"domain"`
	GeoSite   []string `yaml:"geosite" json:"geosite"`
}

// DNS config
type DNS struct {
	Enable                bool             `yaml:"enable"`
	PreferH3              bool             `yaml:"prefer-h3"`
	IPv6                  bool             `yaml:"ipv6"`
	IPv6Timeout           uint             `yaml:"ipv6-timeout"`
	UseSystemHosts        bool             `yaml:"use-system-hosts"`
	NameServer            []dns.NameServer `yaml:"nameserver"`
	Fallback              []dns.NameServer `yaml:"fallback"`
	FallbackFilter        FallbackFilter   `yaml:"fallback-filter"`
	Listen                string           `yaml:"listen"`
	EnhancedMode          C.DNSMode        `yaml:"enhanced-mode"`
	DefaultNameserver     []dns.NameServer `yaml:"default-nameserver"`
	CacheAlgorithm        string           `yaml:"cache-algorithm"`
	FakeIPRange           *fakeip.Pool
	Hosts                 *trie.DomainTrie[resolver.HostValue]
	NameServerPolicy      *orderedmap.OrderedMap[string, []dns.NameServer]
	ProxyServerNameserver []dns.NameServer
}
