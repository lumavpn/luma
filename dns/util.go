package dns

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"

	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/nnip"
	"github.com/lumavpn/luma/common/picker"
	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/dns/resolver"
	"github.com/lumavpn/luma/log"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	C "github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxydialer"

	D "github.com/miekg/dns"
	"github.com/samber/lo"
)

const (
	MaxMsgSize = 65535
)

const serverFailureCacheTTL uint32 = 5

func minimalTTL(records []D.RR) uint32 {
	rr := lo.MinBy(records, func(r1 D.RR, r2 D.RR) bool {
		return r1.Header().Ttl < r2.Header().Ttl
	})
	if rr == nil {
		return 0
	}
	return rr.Header().Ttl
}

func updateTTL(records []D.RR, ttl uint32) {
	if len(records) == 0 {
		return
	}
	delta := minimalTTL(records) - ttl
	for i := range records {
		records[i].Header().Ttl = lo.Clamp(records[i].Header().Ttl-delta, 1, records[i].Header().Ttl)
	}
}

func putMsgToCache(c dnsCache, key string, q D.Question, msg *D.Msg) {
	// skip dns cache for acme challenge
	if q.Qtype == D.TypeTXT && strings.HasPrefix(q.Name, "_acme-challenge.") {
		log.Debugf("[DNS] dns cache ignored because of acme challenge for: %s", q.Name)
		return
	}

	var ttl uint32
	if msg.Rcode == D.RcodeServerFailure {
		// [...] a resolver MAY cache a server failure response.
		// If it does so it MUST NOT cache it for longer than five (5) minutes [...]
		ttl = serverFailureCacheTTL
	} else {
		ttl = minimalTTL(append(append(msg.Answer, msg.Ns...), msg.Extra...))
	}
	if ttl == 0 {
		return
	}
	c.SetWithExpire(key, msg.Copy(), time.Now().Add(time.Duration(ttl)*time.Second))
}

func setMsgTTL(msg *D.Msg, ttl uint32) {
	for _, answer := range msg.Answer {
		answer.Header().Ttl = ttl
	}

	for _, ns := range msg.Ns {
		ns.Header().Ttl = ttl
	}

	for _, extra := range msg.Extra {
		extra.Header().Ttl = ttl
	}
}

func updateMsgTTL(msg *D.Msg, ttl uint32) {
	updateTTL(msg.Answer, ttl)
	updateTTL(msg.Ns, ttl)
	updateTTL(msg.Extra, ttl)
}

func isIPRequest(q D.Question) bool {
	return q.Qclass == D.ClassINET && (q.Qtype == D.TypeA || q.Qtype == D.TypeAAAA || q.Qtype == D.TypeCNAME)
}

func transform(servers []NameServer, m proxydialer.ProxyDialer, resolver *Resolver) []dnsClient {
	ret := make([]dnsClient, 0, len(servers))
	for _, s := range servers {
		switch s.Net {
		case "https":
			ret = append(ret, newDoHClient(s.Addr, resolver, s.PreferH3, s.Params, s.ProxyAdapter, m, s.ProxyName))
			continue
		case "dhcp":
			ret = append(ret, newDHCPClient(s.Addr, m))
			continue
		case "system":
			ret = append(ret, newSystemClient(m))
			continue
		case "rcode":
			ret = append(ret, newRCodeClient(s.Addr))
			continue
		case "quic":
			if doq, err := newDoQ(resolver, s.Addr, s.ProxyAdapter, m, s.ProxyName); err == nil {
				ret = append(ret, doq)
			} else {
				log.Fatalf("DoQ format error: %v", err)
			}
			continue
		}

		host, port, _ := net.SplitHostPort(s.Addr)
		ret = append(ret, &client{
			Client: &D.Client{
				Net: s.Net,
				TLSConfig: &tls.Config{
					ServerName: host,
				},
				UDPSize: 4096,
				Timeout: 5 * time.Second,
			},
			port:         port,
			host:         host,
			iface:        s.Interface,
			r:            resolver,
			proxyAdapter: s.ProxyAdapter,
			proxyName:    s.ProxyName,
		})
	}
	return ret
}

func handleMsgWithEmptyAnswer(r *D.Msg) *D.Msg {
	msg := &D.Msg{}
	msg.Answer = []D.RR{}

	msg.SetRcode(r, D.RcodeSuccess)
	msg.Authoritative = true
	msg.RecursionAvailable = true

	return msg
}

func msgToIP(msg *D.Msg) []netip.Addr {
	ips := []netip.Addr{}

	for _, answer := range msg.Answer {
		switch ans := answer.(type) {
		case *D.AAAA:
			ips = append(ips, nnip.IpToAddr(ans.AAAA))
		case *D.A:
			ips = append(ips, nnip.IpToAddr(ans.A))
		}
	}

	return ips
}

func msgToDomain(msg *D.Msg) string {
	if len(msg.Question) > 0 {
		return strings.TrimRight(msg.Question[0].Name, ".")
	}

	return ""
}

type dialHandler func(ctx context.Context, network, addr string) (net.Conn, error)

func getDialHandler(r *Resolver, proxyDialer proxydialer.ProxyDialer, proxyAdapter proxy.ProxyAdapter,
	proxyName string, opts ...dialer.Option) dialHandler {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if len(proxyName) == 0 && proxyAdapter == nil {
			opts = append(opts, dialer.WithResolver(r))
			return dialer.DialContext(ctx, network, addr, opts...)
		} else {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			uintPort, err := strconv.ParseUint(port, 10, 16)
			if err != nil {
				return nil, err
			}
			if proxyAdapter == nil {
				var ok bool
				proxyAdapter, ok = proxyDialer.Proxies()[proxyName]
				if !ok {
					opts = append(opts, dialer.WithInterface(proxyName))
				}
			}

			if strings.Contains(network, "tcp") {
				// tcp can resolve host by remote
				metadata := &M.Metadata{
					Network: M.TCP,
					Host:    host,
					DstPort: uint16(uintPort),
				}
				if proxyAdapter != nil {
					if proxyAdapter.IsL3Protocol(metadata) { // L3 proxy should resolve domain before to avoid loopback
						dstIP, err := resolver.ResolveIPWithResolver(ctx, host, r)
						if err != nil {
							return nil, err
						}
						metadata.Host = ""
						metadata.DstIP = dstIP
					}
					return proxyAdapter.DialContext(ctx, metadata, opts...)
				}
				opts = append(opts, dialer.WithResolver(r))
				return dialer.DialContext(ctx, network, addr, opts...)
			} else {
				// udp must resolve host first
				dstIP, err := resolver.ResolveIPWithResolver(ctx, host, r)
				if err != nil {
					return nil, err
				}
				metadata := &M.Metadata{
					Network: M.UDP,
					Host:    "",
					DstIP:   dstIP,
					DstPort: uint16(uintPort),
				}
				if proxyAdapter == nil {
					return dialer.DialContext(ctx, network, addr, opts...)
				}

				if !proxyAdapter.SupportUDP() {
					return nil, fmt.Errorf("proxy adapter [%s] UDP is not supported", proxyAdapter)
				}

				packetConn, err := proxyAdapter.ListenPacketContext(ctx, metadata, opts...)
				if err != nil {
					return nil, err
				}

				return N.NewBindPacketConn(packetConn, metadata.UDPAddr()), nil
			}
		}
	}
}

func listenPacket(ctx context.Context, proxyDialer proxydialer.ProxyDialer, proxyAdapter C.ProxyAdapter,
	proxyName string, network string, addr string, r *Resolver, opts ...dialer.Option) (net.PacketConn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	uintPort, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return nil, err
	}
	if proxyAdapter == nil {
		var ok bool
		proxyAdapter, ok = proxyDialer.Proxies()[proxyName]
		if !ok {
			opts = append(opts, dialer.WithInterface(proxyName))
		}
	}

	// udp must resolve host first
	dstIP, err := resolver.ResolveIPWithResolver(ctx, host, r)
	if err != nil {
		return nil, err
	}
	metadata := &M.Metadata{
		Network: M.UDP,
		Host:    "",
		DstIP:   dstIP,
		DstPort: uint16(uintPort),
	}
	if proxyAdapter == nil {
		return dialer.NewDialer(opts...).ListenPacket(ctx, network, "", netip.AddrPortFrom(metadata.DstIP, metadata.DstPort))
	}

	if !proxyAdapter.SupportUDP() {
		return nil, fmt.Errorf("proxy adapter [%s] UDP is not supported", proxyAdapter)
	}

	return proxyAdapter.ListenPacketContext(ctx, metadata, opts...)
}

func batchExchange(ctx context.Context, clients []dnsClient, m *D.Msg) (msg *D.Msg, cache bool, err error) {
	cache = true
	fast, ctx := picker.WithTimeout[*D.Msg](ctx, resolver.DefaultDNSTimeout)
	defer fast.Close()
	domain := msgToDomain(m)
	var noIpMsg *D.Msg
	for _, client := range clients {
		if _, isRCodeClient := client.(rcodeClient); isRCodeClient {
			msg, err = client.ExchangeContext(ctx, m)
			return msg, false, err
		}
		client := client // shadow define client to ensure the value captured by the closure will not be changed in the next loop
		fast.Go(func() (*D.Msg, error) {
			log.Debugf("[DNS] resolve %s from %s", domain, client.Address())
			m, err := client.ExchangeContext(ctx, m)
			if err != nil {
				return nil, err
			} else if cache && (m.Rcode == D.RcodeServerFailure || m.Rcode == D.RcodeRefused) {
				// currently, cache indicates whether this msg was from a RCode client,
				// so we would ignore RCode errors from RCode clients.
				return nil, errors.New("server failure: " + D.RcodeToString[m.Rcode])
			}
			if ips := msgToIP(m); len(m.Question) > 0 {
				qType := m.Question[0].Qtype
				log.Debugf("[DNS] %s --> %s %s from %s", domain, ips, D.Type(qType), client.Address())
				switch qType {
				case D.TypeAAAA:
					if len(ips) == 0 {
						noIpMsg = m
						return nil, resolver.ErrIPNotFound
					}
				case D.TypeA:
					if len(ips) == 0 {
						noIpMsg = m
						return nil, resolver.ErrIPNotFound
					}
				}
			}
			return m, nil
		})
	}

	msg = fast.Wait()
	if msg == nil {
		if noIpMsg != nil {
			return noIpMsg, false, nil
		}
		err = errors.New("all DNS requests failed")
		if fErr := fast.Error(); fErr != nil {
			err = fmt.Errorf("%w, first error: %w", err, fErr)
		}
	}
	return
}
