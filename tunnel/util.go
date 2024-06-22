package tunnel

import (
	"context"
	"errors"
	"net/netip"

	"github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/component/resolver"
	"github.com/lumavpn/luma/component/slowdown"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/protos"
	"github.com/lumavpn/luma/rule"
)

func (t *tunnel) isHandle(p protos.Protocol) bool {
	status := t.status.Load()
	return status == Running || (status == Inner && p == protos.Protocol_INNER)
}

func needLookupIP(m *metadata.Metadata) bool {
	return m.Host == "" && m.DstIP.IsValid()
}

func preHandleMetadata(m *metadata.Metadata) error {
	// handle IP string on host
	if ip, err := netip.ParseAddr(m.Host); err == nil {
		m.DstIP = ip
		m.Host = ""
	}
	return nil
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
	default:
		return t.proxyDialer.Match(m)
	}

	return proxy, nil, err
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
	return false
}
