package luma

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"regexp"
	"strings"

	C "github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/component/fakeip"
	"github.com/lumavpn/luma/component/resolver"
	"github.com/lumavpn/luma/component/trie"
	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/dns"
	"github.com/lumavpn/luma/geodata"
	"github.com/lumavpn/luma/geodata/router"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy/provider"
	"github.com/lumavpn/luma/rule"
	ruleTypes "github.com/lumavpn/luma/rule"
	"github.com/lumavpn/luma/tunnel"
	"github.com/lumavpn/luma/util"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func hostWithDefaultPort(host string, defPort string) (string, error) {
	hostname, port, err := net.SplitHostPort(host)
	if err != nil {
		if !strings.Contains(err.Error(), "missing port in address") {
			return "", err
		}
		host = host + ":" + defPort
		if hostname, port, err = net.SplitHostPort(host); err != nil {
			return "", err
		}
	}

	return net.JoinHostPort(hostname, port), nil
}

func parseNameServer(servers []string, preferH3 bool) ([]dns.NameServer, error) {
	var nameservers []dns.NameServer

	for idx, server := range servers {
		server = parsePureDNSServer(server)
		u, err := url.Parse(server)
		if err != nil {
			return nil, fmt.Errorf("DNS NameServer[%d] format error: %s", idx, err.Error())
		}

		proxyName := u.Fragment

		var addr, dnsNetType string
		params := map[string]string{}
		switch u.Scheme {
		case "udp":
			addr, err = hostWithDefaultPort(u.Host, "53")
			dnsNetType = "" // UDP
		case "tcp":
			addr, err = hostWithDefaultPort(u.Host, "53")
			dnsNetType = "tcp" // TCP
		case "tls":
			addr, err = hostWithDefaultPort(u.Host, "853")
			dnsNetType = "tcp-tls" // DNS over TLS
		case "https":
			addr, err = hostWithDefaultPort(u.Host, "443")
			if err == nil {
				proxyName = ""
				clearURL := url.URL{Scheme: "https", Host: addr, Path: u.Path, User: u.User}
				addr = clearURL.String()
				dnsNetType = "https" // DNS over HTTPS
				if len(u.Fragment) != 0 {
					for _, s := range strings.Split(u.Fragment, "&") {
						arr := strings.Split(s, "=")
						if len(arr) == 0 {
							continue
						} else if len(arr) == 1 {
							proxyName = arr[0]
						} else if len(arr) == 2 {
							params[arr[0]] = arr[1]
						} else {
							params[arr[0]] = strings.Join(arr[1:], "=")
						}
					}
				}
			}
		case "dhcp":
			addr = u.Host
			dnsNetType = "dhcp" // UDP from DHCP
		case "quic":
			addr, err = hostWithDefaultPort(u.Host, "853")
			dnsNetType = "quic" // DNS over QUIC
		case "system":
			dnsNetType = "system" // System DNS
		case "rcode":
			dnsNetType = "rcode"
			addr = u.Host
			switch addr {
			case "success",
				"format_error",
				"server_failure",
				"name_error",
				"not_implemented",
				"refused":
			default:
				err = fmt.Errorf("unsupported RCode type: %s", addr)
			}
		default:
			return nil, fmt.Errorf("DNS NameServer[%d] unsupport scheme: %s", idx, u.Scheme)
		}

		if err != nil {
			return nil, fmt.Errorf("DNS NameServer[%d] format error: %s", idx, err.Error())
		}

		nameservers = append(
			nameservers,
			dns.NameServer{
				Net:       dnsNetType,
				Addr:      addr,
				ProxyName: proxyName,
				Params:    params,
				PreferH3:  preferH3,
			},
		)
	}
	return nameservers, nil
}

func parsePureDNSServer(server string) string {
	addPre := func(server string) string {
		return "udp://" + server
	}

	if server == "system" {
		return "system://"
	}

	if ip, err := netip.ParseAddr(server); err != nil {
		if strings.Contains(server, "://") {
			return server
		}
		return addPre(server)
	} else {
		if ip.Is4() {
			return addPre(server)
		} else {
			return addPre("[" + server + "]")
		}
	}
}
func parseNameServerPolicy(nsPolicy *orderedmap.OrderedMap[string, any], ruleProviders map[string]provider.RuleProvider, preferH3 bool) (*orderedmap.OrderedMap[string, []dns.NameServer], error) {
	policy := orderedmap.New[string, []dns.NameServer]()
	updatedPolicy := orderedmap.New[string, any]()
	re := regexp.MustCompile(`[a-zA-Z0-9\-]+\.[a-zA-Z]{2,}(\.[a-zA-Z]{2,})?`)

	for pair := nsPolicy.Oldest(); pair != nil; pair = pair.Next() {
		k, v := pair.Key, pair.Value
		if strings.Contains(strings.ToLower(k), ",") {
			if strings.Contains(k, "geosite:") {
				subkeys := strings.Split(k, ":")
				subkeys = subkeys[1:]
				subkeys = strings.Split(subkeys[0], ",")
				for _, subkey := range subkeys {
					newKey := "geosite:" + subkey
					updatedPolicy.Store(newKey, v)
				}
			} else if strings.Contains(strings.ToLower(k), "rule-set:") {
				subkeys := strings.Split(k, ":")
				subkeys = subkeys[1:]
				subkeys = strings.Split(subkeys[0], ",")
				for _, subkey := range subkeys {
					newKey := "rule-set:" + subkey
					updatedPolicy.Store(newKey, v)
				}
			} else if re.MatchString(k) {
				subkeys := strings.Split(k, ",")
				for _, subkey := range subkeys {
					updatedPolicy.Store(subkey, v)
				}
			}
		} else {
			if strings.Contains(strings.ToLower(k), "geosite:") {
				updatedPolicy.Store("geosite:"+k[8:], v)
			} else if strings.Contains(strings.ToLower(k), "rule-set:") {
				updatedPolicy.Store("rule-set:"+k[9:], v)
			}
			updatedPolicy.Store(k, v)
		}
	}

	for pair := updatedPolicy.Oldest(); pair != nil; pair = pair.Next() {
		domain, server := pair.Key, pair.Value
		servers, err := util.ToStringSlice(server)
		if err != nil {
			return nil, err
		}
		nameservers, err := parseNameServer(servers, preferH3)
		if err != nil {
			return nil, err
		}
		if _, valid := trie.ValidAndSplitDomain(domain); !valid {
			return nil, fmt.Errorf("DNS ResoverRule invalid domain: %s", domain)
		}
		if strings.HasPrefix(domain, "rule-set:") {
			domainSetName := domain[9:]
			if ruleProvider, ok := ruleProviders[domainSetName]; !ok {
				return nil, fmt.Errorf("not found rule-set: %s", domainSetName)
			} else {
				switch ruleProvider.Behavior() {
				case provider.IPCIDR:
					return nil, fmt.Errorf("rule provider type error, except domain,actual %s", ruleProvider.Behavior())
				case provider.Classical:
					log.Warnf("%s provider is %s, only matching it contain domain rule", ruleProvider.Name(),
						ruleProvider.Behavior())
				}
			}
		}
		policy.Store(domain, nameservers)
	}

	return policy, nil
}

func parseFallbackIPCIDR(ips []string) ([]netip.Prefix, error) {
	var ipNets []netip.Prefix

	for idx, ip := range ips {
		ipnet, err := netip.ParsePrefix(ip)
		if err != nil {
			return nil, fmt.Errorf("DNS FallbackIP[%d] format error: %s", idx, err.Error())
		}
		ipNets = append(ipNets, ipnet)
	}

	return ipNets, nil
}

func parseFallbackGeoSite(countries []string, rules []rule.Rule) ([]router.DomainMatcher, error) {
	var sites []router.DomainMatcher
	if len(countries) > 0 {
		if err := geodata.InitGeoSite(); err != nil {
			return nil, fmt.Errorf("can't initial GeoSite: %s", err)
		}
		log.Warnf("replace fallback-filter.geosite with nameserver-policy, it will be removed in the future")
	}

	for _, country := range countries {
		found := false
		for _, rule := range rules {
			if rule.RuleType() == ruleTypes.GEOSITE {
				if strings.EqualFold(country, rule.Payload()) {
					found = true
					sites = append(sites, rule.(ruleTypes.RuleGeoSite).GetDomainMatcher())
					log.Infof("Start initial GeoSite dns fallback filter from rule `%s`", country)
				}
			}
		}

		if !found {
			matcher, recordsCount, err := geodata.LoadGeoSiteMatcher(country)
			if err != nil {
				return nil, err
			}

			sites = append(sites, matcher)

			log.Infof("Start initial GeoSite dns fallback filter `%s`, records: %d", country, recordsCount)
		}
	}
	return sites, nil
}

func parseDNS(ccfg *config.Config, tunnel tunnel.Tunnel, hosts *trie.DomainTrie[resolver.HostValue], rules []rule.Rule,
	ruleProviders map[string]provider.RuleProvider) (*config.DNS, error) {
	cfg := ccfg.RawDNS
	if cfg.Enable && len(cfg.NameServer) == 0 {
		return nil, fmt.Errorf("if DNS configuration is turned on, NameServer cannot be empty")
	}

	dnsCfg := &config.DNS{
		Enable:       cfg.Enable,
		Listen:       cfg.Listen,
		PreferH3:     cfg.PreferH3,
		IPv6Timeout:  cfg.IPv6Timeout,
		IPv6:         cfg.IPv6,
		EnhancedMode: cfg.EnhancedMode,
		FallbackFilter: config.FallbackFilter{
			IPCIDR:  []netip.Prefix{},
			GeoSite: []router.DomainMatcher{},
		},
	}
	var err error
	if dnsCfg.NameServer, err = parseNameServer(cfg.NameServer, cfg.PreferH3); err != nil {
		return nil, err
	}

	if dnsCfg.Fallback, err = parseNameServer(cfg.Fallback, cfg.PreferH3); err != nil {
		return nil, err
	}

	if dnsCfg.NameServerPolicy, err = parseNameServerPolicy(cfg.NameServerPolicy, ruleProviders, cfg.PreferH3); err != nil {
		return nil, err
	}

	if dnsCfg.ProxyServerNameserver, err = parseNameServer(cfg.ProxyServerNameserver, cfg.PreferH3); err != nil {
		return nil, err
	}

	if len(cfg.DefaultNameserver) == 0 {
		return nil, errors.New("default nameserver should have at least one nameserver")
	}
	if dnsCfg.DefaultNameserver, err = parseNameServer(cfg.DefaultNameserver, cfg.PreferH3); err != nil {
		return nil, err
	}
	// check default nameserver is pure ip addr
	for _, ns := range dnsCfg.DefaultNameserver {
		if ns.Net == "system" {
			continue
		}
		host, _, err := net.SplitHostPort(ns.Addr)
		if err != nil || net.ParseIP(host) == nil {
			u, err := url.Parse(ns.Addr)
			if err == nil && net.ParseIP(u.Host) == nil {
				if ip, _, err := net.SplitHostPort(u.Host); err != nil || net.ParseIP(ip) == nil {
					return nil, errors.New("default nameserver should be pure IP")
				}
			}
		}
	}

	fakeIPRange, err := netip.ParsePrefix(cfg.FakeIPRange)
	tunnel.SetFakeIPRange(fakeIPRange)
	if cfg.EnhancedMode == C.DNSFakeIP {
		if err != nil {
			return nil, err
		}

		var host *trie.DomainTrie[struct{}]
		// fake ip skip host filter
		if len(cfg.FakeIPFilter) != 0 {
			host = trie.New[struct{}]()
			for _, domain := range cfg.FakeIPFilter {
				_ = host.Insert(domain, struct{}{})
			}
			host.Optimize()
		}

		if len(dnsCfg.Fallback) != 0 {
			if host == nil {
				host = trie.New[struct{}]()
			}
			for _, fb := range dnsCfg.Fallback {
				if net.ParseIP(fb.Addr) != nil {
					continue
				}
				_ = host.Insert(fb.Addr, struct{}{})
			}
			host.Optimize()
		}

		pool, err := fakeip.New(fakeip.Options{
			IPNet:       fakeIPRange,
			Size:        1000,
			Host:        host,
			Persistence: ccfg.Profile.StoreFakeIP,
		})
		if err != nil {
			return nil, err
		}

		dnsCfg.FakeIPRange = pool
	}

	if len(cfg.Fallback) != 0 {
		dnsCfg.FallbackFilter.GeoIP = cfg.FallbackFilter.GeoIP
		dnsCfg.FallbackFilter.GeoIPCode = cfg.FallbackFilter.GeoIPCode
		if fallbackip, err := parseFallbackIPCIDR(cfg.FallbackFilter.IPCIDR); err == nil {
			dnsCfg.FallbackFilter.IPCIDR = fallbackip
		}
		dnsCfg.FallbackFilter.Domain = cfg.FallbackFilter.Domain
		fallbackGeoSite, err := parseFallbackGeoSite(cfg.FallbackFilter.GeoSite, rules)
		if err != nil {
			return nil, fmt.Errorf("load GeoSite dns fallback filter error, %w", err)
		}
		dnsCfg.FallbackFilter.GeoSite = fallbackGeoSite
	}

	if cfg.UseHosts {
		dnsCfg.Hosts = hosts
	}

	if cfg.CacheAlgorithm == "" || cfg.CacheAlgorithm == "lru" {
		dnsCfg.CacheAlgorithm = "lru"
	} else {
		dnsCfg.CacheAlgorithm = "arc"
	}

	return dnsCfg, nil
}

func parseHosts(hosts map[string]any) (*trie.DomainTrie[resolver.HostValue], error) {
	tree := trie.New[resolver.HostValue]()

	// add default hosts
	hostValue, _ := resolver.NewHostValueByIPs(
		[]netip.Addr{netip.AddrFrom4([4]byte{127, 0, 0, 1})})
	if err := tree.Insert("localhost", hostValue); err != nil {
		log.Errorf("insert localhost to host error: %s", err.Error())
	}

	if len(hosts) != 0 {
		for domain, anyValue := range hosts {
			if str, ok := anyValue.(string); ok && str == "lan" {
				if addrs, err := net.InterfaceAddrs(); err != nil {
					log.Errorf("insert lan to host error: %s", err)
				} else {
					ips := make([]netip.Addr, 0)
					for _, addr := range addrs {
						if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && !ipnet.IP.IsLinkLocalUnicast() {
							if ip, err := netip.ParseAddr(ipnet.IP.String()); err == nil {
								ips = append(ips, ip)
							}
						}
					}
					anyValue = ips
				}
			}
			value, err := resolver.NewHostValue(anyValue)
			if err != nil {
				return nil, fmt.Errorf("%s is not a valid value", anyValue)
			}
			if value.IsDomain {
				node := tree.Search(value.Domain)
				for node != nil && node.Data().IsDomain {
					if node.Data().Domain == domain {
						return nil, fmt.Errorf("%s, there is a cycle in domain name mapping", domain)
					}
					node = tree.Search(node.Data().Domain)
				}
			}
			_ = tree.Insert(domain, value)
		}
	}
	tree.Optimize()

	return tree, nil
}
