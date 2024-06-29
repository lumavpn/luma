package luma

import (
	"fmt"
	"net"
	"net/netip"

	"github.com/lumavpn/luma/component/trie"
	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/dns/resolver"
	"github.com/lumavpn/luma/log"
)

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

func (lu *Luma) updateDNS(cfg *config.DNSConfig) error {
	if cfg == nil || !cfg.Enable {
		resolver.DefaultResolver = nil
		return nil
	}

	resolver.DisableIPv6 = !cfg.IPv6

	hosts, err := parseHosts(cfg.Hosts)
	if err != nil {
		return err
	}

	resolver.DefaultHosts = resolver.NewHosts(hosts)
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
