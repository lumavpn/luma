package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"

	"github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/component/loopback"
	"github.com/lumavpn/luma/component/slowdown"
	"github.com/lumavpn/luma/dns/resolver"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/rule"
)

type ReadOnlyReader struct {
	io.Reader
}

type WriteOnlyWriter struct {
	io.Writer
}

func needLookupIP(metadata *metadata.Metadata) bool {
	return resolver.MappingEnabled() && metadata.Host == "" && metadata.DstIP.IsValid()
}

func (t *tunnel) resolveMetadata(m *metadata.Metadata) (proxy.Proxy, rule.Rule, error) {
	var err error
	if m.SpecialProxy != "" {
		proxy, err := t.proxyDialer.SelectProxyByName(m.SpecialProxy)
		return proxy, nil, err
	}

	var proxy proxy.Proxy
	mode := t.mode
	switch mode {
	case common.Direct:
		proxy, err = t.proxyDialer.SelectProxyByName("DIRECT")
	case common.Global:
		proxy, err = t.proxyDialer.SelectProxyByName("GLOBAL")
	case common.Select:
		proxy, err = t.proxyDialer.SelectProxy(m)
	default:
		return t.proxyDialer.Match(m)
	}

	return proxy, nil, err
}

func preHandleMetadata(m *metadata.Metadata) error {
	// handle IP string on host
	if ip, err := netip.ParseAddr(m.Host); err == nil {
		m.DstIP = ip
		m.Host = ""
	}
	if needLookupIP(m) {
		host, exist := resolver.FindHostByIP(m.DstIP)
		if exist {
			m.Host = host
			m.DNSMode = common.DNSMapping
			if resolver.FakeIPEnabled() {
				m.DstIP = netip.Addr{}
				m.DNSMode = common.DNSFakeIP
			} else if node, ok := resolver.DefaultHosts.Search(host, false); ok {
				// redir-host should lookup the hosts
				m.DstIP, _ = node.RandIP()
			} else if node != nil && node.IsDomain {
				m.Host = node.Domain
			}
		} else if resolver.IsFakeIP(m.DstIP) {
			return fmt.Errorf("fake DNS record %s missing", m.DstIP)
		}
	} else if node, ok := resolver.DefaultHosts.Search(m.Host, true); ok {
		// try use domain mapping
		m.Host = node.Domain
	}

	return nil
}

func shouldStopRetry(err error) bool {
	if errors.Is(err, resolver.ErrIPNotFound) {
		return true
	}
	if errors.Is(err, resolver.ErrIPVersion) {
		return true
	}
	if errors.Is(err, resolver.ErrIPv6Disabled) {
		return true
	}
	if errors.Is(err, loopback.ErrReject) {
		return true
	}
	return false
}

func retry[T any](ctx context.Context, ft func(context.Context) (T, error), fe func(err error)) (t T, err error) {
	s := slowdown.New()
	for i := 0; i < 10; i++ {
		t, err = ft(ctx)
		if err != nil {
			if fe != nil {
				fe(err)
			}
			if shouldStopRetry(err) {
				return
			}
			if s.Wait(ctx) == nil {
				continue
			} else {
				return
			}
		} else {
			break
		}
	}
	return
}

// handleSocket copies between left and right bidirectionally.
func handleSocket(leftConn, rightConn net.Conn) {
	network.Relay(leftConn, rightConn)
}
