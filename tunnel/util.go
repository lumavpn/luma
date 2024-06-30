package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"time"

	"github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/component/slowdown"
	"github.com/lumavpn/luma/dns/resolver"
	"github.com/lumavpn/luma/metadata"
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
	if errors.Is(err, common.ErrRejectLoopback) {
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
	ch := make(chan error)

	go func() {
		_, err := io.Copy(WriteOnlyWriter{Writer: leftConn}, ReadOnlyReader{Reader: rightConn})
		leftConn.SetReadDeadline(time.Now())
		ch <- err
	}()

	io.Copy(WriteOnlyWriter{Writer: rightConn}, ReadOnlyReader{Reader: leftConn})
	rightConn.SetReadDeadline(time.Now())
	<-ch
}
