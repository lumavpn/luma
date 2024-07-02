package luma

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"

	"github.com/lumavpn/luma/component/trie"
	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/dns"
	"github.com/lumavpn/luma/dns/resolver"
	"github.com/lumavpn/luma/geodata/router"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy/provider"
)

type dnsResult struct {
	config            *config.DNS
	hosts             *trie.DomainTrie[resolver.HostValue]
	defaultNameServer []dns.NameServer
	nameServer        []dns.NameServer
}

func (lu *Luma) shouldHijackDns(targetAddr netip.AddrPort) bool {
	if targetAddr.Addr().IsLoopback() && targetAddr.Port() == 53 { // cause by system stack
		return true
	}
	for _, addrPort := range lu.dnsAdds {
		if addrPort == targetAddr || (addrPort.Addr().IsUnspecified() && targetAddr.Port() == 53) {
			return true
		}
	}
	return false
}

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

func parseNameServer(servers []string, preferH3 bool) ([]dns.NameServer, error) {
	var nameservers []dns.NameServer
	for _, server := range servers {
		server = parsePureDNSServer(server)
		u, err := url.Parse(server)
		if err != nil {
			return nil, fmt.Errorf("DNS NameServer format error: %s", err.Error())
		}

		var addr, dnsNetType string
		switch u.Scheme {
		case "udp":
			addr, err = hostWithDefaultPort(u.Host, "53")
			dnsNetType = ""
		case "system":
			dnsNetType = "system"
		default:
			return nil, fmt.Errorf("DNS NameServer unsupport scheme: %s", u.Scheme)
		}
		if err != nil {
			return nil, fmt.Errorf("DNS NameServer format error: %s", err.Error())
		}

		nameservers = append(
			nameservers,
			dns.NameServer{
				Net:  dnsNetType,
				Addr: addr,
			},
		)
	}
	return nameservers, nil
}

func (lu *Luma) parseDNS(ccfg *config.Config) (result *dnsResult, err error) {
	cfg := ccfg.RawDNS
	dnsConfig := &config.DNS{
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

	result = &dnsResult{
		config: dnsConfig,
	}
	result.hosts, err = parseHosts(ccfg.Hosts)
	if err != nil {
		return
	}
	lu.mu.Lock()
	lu.hosts = result.hosts
	lu.mu.Unlock()

	if cfg.Enable && len(cfg.NameServer) == 0 {
		return nil, fmt.Errorf("name server cannot be empty")
	}
	if dnsConfig.NameServer, err = parseNameServer(cfg.NameServer, cfg.PreferH3); err != nil {
		return nil, err
	}
	if len(cfg.DefaultNameserver) == 0 {
		return nil, errors.New("default nameserver should have at least one nameserver")
	}
	if dnsConfig.DefaultNameserver, err = parseNameServer(cfg.DefaultNameserver, cfg.PreferH3); err != nil {
		return nil, err
	}
	for _, ns := range dnsConfig.DefaultNameserver {
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

	return
}

func (lu *Luma) updateDNS(c *config.DNS, ruleProvider map[string]provider.RuleProvider) error {
	if !c.Enable {
		resolver.DefaultResolver = nil
		resolver.DefaultHostMapper = nil
		resolver.DefaultLocalServer = nil
		dns.ReCreateServer("", nil, nil)
		return nil
	}
	log.Debug("Updating dns")
	cfg := dns.Config{
		Main:         c.NameServer,
		Fallback:     c.Fallback,
		IPv6:         c.IPv6,
		IPv6Timeout:  c.IPv6Timeout,
		EnhancedMode: c.EnhancedMode,
		Pool:         c.FakeIPRange,
		Hosts:        c.Hosts,
		FallbackFilter: dns.FallbackFilter{
			GeoIP:     c.FallbackFilter.GeoIP,
			GeoIPCode: c.FallbackFilter.GeoIPCode,
			IPCIDR:    c.FallbackFilter.IPCIDR,
			Domain:    c.FallbackFilter.Domain,
			GeoSite:   c.FallbackFilter.GeoSite,
		},
		Default:        c.DefaultNameserver,
		Policy:         c.NameServerPolicy,
		ProxyServer:    c.ProxyServerNameserver,
		RuleProviders:  ruleProvider,
		CacheAlgorithm: c.CacheAlgorithm,
	}

	r := dns.NewResolver(cfg, lu.proxyDialer)
	pr := dns.NewProxyServerHostResolver(r)
	m := dns.NewEnhancer(cfg)

	// reuse cache of old host mapper
	if old := resolver.DefaultHostMapper; old != nil {
		m.PatchFrom(old.(*dns.ResolverEnhancer))
	}

	resolver.DefaultResolver = r
	resolver.DefaultHostMapper = m
	resolver.DefaultLocalServer = dns.NewLocalServer(r, m)
	resolver.UseSystemHosts = c.UseSystemHosts

	if pr.Invalid() {
		resolver.ProxyServerHostResolver = pr
	}

	dns.ReCreateServer(c.Listen, r, m)
	return nil
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
