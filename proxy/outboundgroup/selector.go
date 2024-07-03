package outboundgroup

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/lumavpn/luma/dialer"
	M "github.com/lumavpn/luma/metadata"
	C "github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/outbound"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/proxy/provider"
	"github.com/lumavpn/luma/proxydialer"
)

type Selector struct {
	*GroupBase
	disableUDP bool
	selected   string
	Hidden     bool
	Icon       string
}

// DialContext implements C.ProxyAdapter
func (s *Selector) DialContext(ctx context.Context, metadata *M.Metadata, opts ...dialer.Option) (C.Conn, error) {
	c, err := s.selectedProxy(true).DialContext(ctx, metadata, s.Base.DialOptions(opts...)...)
	if err == nil {
		c.AppendToChains(s)
	}
	return c, err
}

// ListenPacketContext implements C.ProxyAdapter
func (s *Selector) ListenPacketContext(ctx context.Context, metadata *M.Metadata, opts ...dialer.Option) (C.PacketConn, error) {
	pc, err := s.selectedProxy(true).ListenPacketContext(ctx, metadata, s.Base.DialOptions(opts...)...)
	if err == nil {
		pc.AppendToChains(s)
	}
	return pc, err
}

// SupportUDP implements C.ProxyAdapter
func (s *Selector) SupportUDP() bool {
	if s.disableUDP {
		return false
	}

	return s.selectedProxy(false).SupportUDP()
}

// IsL3Protocol implements C.ProxyAdapter
func (s *Selector) IsL3Protocol(metadata *M.Metadata) bool {
	return s.selectedProxy(false).IsL3Protocol(metadata)
}

// MarshalJSON implements C.ProxyAdapter
func (s *Selector) MarshalJSON() ([]byte, error) {
	all := []string{}
	for _, proxy := range s.GetProxies(false) {
		all = append(all, proxy.Name())
	}

	return json.Marshal(map[string]any{
		"type":   s.Proto().String(),
		"now":    s.Now(),
		"all":    all,
		"hidden": s.Hidden,
		"icon":   s.Icon,
	})
}

func (s *Selector) Now() string {
	return s.selectedProxy(false).Name()
}

func (s *Selector) Set(name string) error {
	for _, proxy := range s.GetProxies(false) {
		if proxy.Name() == name {
			s.selected = name
			return nil
		}
	}

	return errors.New("proxy not exist")
}

func (s *Selector) ForceSet(name string) {
	s.selected = name
}

// Unwrap implements C.ProxyAdapter
func (s *Selector) Unwrap(metadata *M.Metadata, touch bool) C.Proxy {
	return s.selectedProxy(touch)
}

func (s *Selector) selectedProxy(touch bool) C.Proxy {
	proxies := s.GetProxies(touch)
	for _, proxy := range proxies {
		if proxy.Name() == s.selected {
			return proxy
		}
	}

	return proxies[0]
}

func NewSelector(option *GroupCommonOption, proxyDialer proxydialer.ProxyDialer, providers []provider.ProxyProvider) *Selector {
	return &Selector{
		GroupBase: NewGroupBase(GroupBaseOption{
			outbound.BaseOption{
				Name:        option.Name,
				Proto:       proto.Proto_Selector,
				Interface:   option.Interface,
				RoutingMark: option.RoutingMark,
			},
			option.Filter,
			option.ExcludeFilter,
			option.ExcludeType,
			option.TestTimeout,
			option.MaxFailedTimes,
			providers,
		}, proxyDialer),
		selected:   "COMPATIBLE",
		disableUDP: option.DisableUDP,
		Hidden:     option.Hidden,
		Icon:       option.Icon,
	}
}
