package dns

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/lumavpn/luma/common/picker"
	"github.com/lumavpn/luma/dns/resolver"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/util"
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
	if q.Qtype == D.TypeTXT && strings.HasPrefix(q.Name, "_acme-challenge.") {
		log.Debugf("[DNS] dns cache ignored because of acme challenge for: %s", q.Name)
		return
	}

	var ttl uint32
	if msg.Rcode == D.RcodeServerFailure {
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

type nameServerClient struct {
	NameServer
	dnsClient
}

func cacheTransform(defaultResolver *Resolver, servers []NameServer) []dnsClient {
	var nameServerCache []nameServerClient
	var result []dnsClient
	for _, ns := range servers {
		for _, nsc := range nameServerCache {
			if nsc.NameServer.Equal(ns) {
				result = append(result, nsc.dnsClient)
				break
			}
		}
		// not in cache
		dc := transform([]NameServer{ns}, defaultResolver)
		if len(dc) > 0 {
			dc := dc[0]
			nameServerCache = append(nameServerCache, struct {
				NameServer
				dnsClient
			}{NameServer: ns, dnsClient: dc})
			result = append(result, dc)
		}
	}
	return result
}

func transform(servers []NameServer, resolver *Resolver) []dnsClient {
	ret := make([]dnsClient, 0, len(servers))
	for _, s := range servers {
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
			port:  port,
			host:  host,
			iface: s.Interface,
			r:     resolver,
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
			ips = append(ips, util.IpToAddr(ans.AAAA))
		case *D.A:
			ips = append(ips, util.IpToAddr(ans.A))
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
		client := client
		fast.Go(func() (*D.Msg, error) {
			log.Debugf("[DNS] resolve %s from %s", domain, client.Address())
			m, err := client.ExchangeContext(ctx, m)
			if err != nil {
				return nil, err
			} else if cache && (m.Rcode == D.RcodeServerFailure || m.Rcode == D.RcodeRefused) {
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
