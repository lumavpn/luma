package outboundgroup

import (
	"context"
	"encoding/json"

	PD "github.com/lumavpn/luma/component/proxydialer"
	"github.com/lumavpn/luma/dialer"
	M "github.com/lumavpn/luma/metadata"
	C "github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/outbound"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/proxy/provider"
	"github.com/lumavpn/luma/proxydialer"
)

type Relay struct {
	*GroupBase
	Hidden bool
	Icon   string
}

// DialContext implements C.ProxyAdapter
func (r *Relay) DialContext(ctx context.Context, metadata *M.Metadata, opts ...dialer.Option) (C.Conn, error) {
	proxies, chainProxies := r.proxies(metadata, true)

	switch len(proxies) {
	case 0:
		return outbound.NewDirect().DialContext(ctx, metadata, r.Base.DialOptions(opts...)...)
	case 1:
		return proxies[0].DialContext(ctx, metadata, r.Base.DialOptions(opts...)...)
	}
	var d C.Dialer
	d = dialer.NewDialer(r.Base.DialOptions(opts...)...)
	for _, proxy := range proxies[:len(proxies)-1] {
		d = PD.New(proxy, d, false)
	}
	last := proxies[len(proxies)-1]
	conn, err := last.DialContextWithDialer(ctx, d, metadata)
	if err != nil {
		return nil, err
	}

	for i := len(chainProxies) - 2; i >= 0; i-- {
		conn.AppendToChains(chainProxies[i])
	}

	conn.AppendToChains(r)

	return conn, nil
}

// ListenPacketContext implements C.ProxyAdapter
func (r *Relay) ListenPacketContext(ctx context.Context, metadata *M.Metadata, opts ...dialer.Option) (_ C.PacketConn, err error) {
	proxies, chainProxies := r.proxies(metadata, true)

	switch len(proxies) {
	case 0:
		return outbound.NewDirect().ListenPacketContext(ctx, metadata, r.Base.DialOptions(opts...)...)
	case 1:
		return proxies[0].ListenPacketContext(ctx, metadata, r.Base.DialOptions(opts...)...)
	}

	var d C.Dialer
	d = dialer.NewDialer(r.Base.DialOptions(opts...)...)
	for _, proxy := range proxies[:len(proxies)-1] {
		d = PD.New(proxy, d, false)
	}
	last := proxies[len(proxies)-1]
	pc, err := last.ListenPacketWithDialer(ctx, d, metadata)
	if err != nil {
		return nil, err
	}

	for i := len(chainProxies) - 2; i >= 0; i-- {
		pc.AppendToChains(chainProxies[i])
	}

	pc.AppendToChains(r)

	return pc, nil
}

// SupportUDP implements C.ProxyAdapter
func (r *Relay) SupportUDP() bool {
	proxies, _ := r.proxies(nil, false)
	if len(proxies) == 0 { // C.Direct
		return true
	}
	for i := len(proxies) - 1; i >= 0; i-- {
		proxy := proxies[i]
		if !proxy.SupportUDP() {
			return false
		}
		if proxy.SupportUOT() {
			return true
		}
		switch proxy.SupportWithDialer() {
		case M.ALLNet:
		case M.UDP:
		default: // C.TCP and C.InvalidNet
			return false
		}
	}
	return true
}

// MarshalJSON implements C.ProxyAdapter
func (r *Relay) MarshalJSON() ([]byte, error) {
	all := []string{}
	for _, proxy := range r.GetProxies(false) {
		all = append(all, proxy.Name())
	}
	return json.Marshal(map[string]any{
		"type":   r.Proto().String(),
		"all":    all,
		"hidden": r.Hidden,
		"icon":   r.Icon,
	})
}

func (r *Relay) proxies(metadata *M.Metadata, touch bool) ([]C.Proxy, []C.Proxy) {
	rawProxies := r.GetProxies(touch)

	var proxies []C.Proxy
	var chainProxies []C.Proxy
	var targetProxies []C.Proxy

	for n, proxy := range rawProxies {
		proxies = append(proxies, proxy)
		chainProxies = append(chainProxies, proxy)
		subproxy := proxy.Unwrap(metadata, touch)
		for subproxy != nil {
			chainProxies = append(chainProxies, subproxy)
			proxies[n] = subproxy
			subproxy = subproxy.Unwrap(metadata, touch)
		}
	}

	for _, proxy := range proxies {
		if proxy.Proto() != proto.Proto_Direct && proxy.Proto() != proto.Proto_Compatible {
			targetProxies = append(targetProxies, proxy)
		}
	}

	return targetProxies, chainProxies
}

func (r *Relay) Addr() string {
	proxies, _ := r.proxies(nil, false)
	return proxies[len(proxies)-1].Addr()
}

func NewRelay(option *GroupCommonOption, proxyDialer proxydialer.ProxyDialer, providers []provider.ProxyProvider) *Relay {
	return &Relay{
		GroupBase: NewGroupBase(GroupBaseOption{
			outbound.BaseOption{
				Name:        option.Name,
				Proto:       proto.Proto_Relay,
				Interface:   option.Interface,
				RoutingMark: option.RoutingMark,
			},
			"",
			"",
			"",
			5000,
			5,
			providers,
		}, proxyDialer),
		Hidden: option.Hidden,
		Icon:   option.Icon,
	}
}
