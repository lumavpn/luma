package outbound

import (
	"context"
	"encoding/json"

	C "github.com/lumavpn/luma/common"
	E "github.com/lumavpn/luma/common/errors"
	"github.com/lumavpn/luma/dialer"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	P "github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/util"
)

type Base struct {
	name        string
	addr        string
	iface       string
	proto       proto.Proto
	udp         bool
	xudp        bool
	tfo         bool
	disableIPv6 bool
	username    string
	password    string
	rmark       int
	mpTcp       bool
	id          string
	prefer      C.DNSPrefer
}

type BaseOption struct {
	Name        string
	Addr        string
	Proto       proto.Proto
	UDP         bool
	XUDP        bool
	TFO         bool
	MPTCP       bool
	Interface   string
	RoutingMark int
	Prefer      C.DNSPrefer
}

func NewBase(opt BaseOption) *Base {
	return &Base{
		name:   opt.Name,
		addr:   opt.Addr,
		proto:  opt.Proto,
		udp:    opt.UDP,
		xudp:   opt.XUDP,
		tfo:    opt.TFO,
		mpTcp:  opt.MPTCP,
		iface:  opt.Interface,
		rmark:  opt.RoutingMark,
		prefer: opt.Prefer,
	}
}

type BasicOption struct {
	TFO         bool               `proxy:"tfo,omitempty" group:"tfo,omitempty"`
	MPTCP       bool               `proxy:"mptcp,omitempty" group:"mptcp,omitempty"`
	Interface   string             `proxy:"interface-name,omitempty" group:"interface-name,omitempty"`
	RoutingMark int                `proxy:"routing-mark,omitempty" group:"routing-mark,omitempty"`
	IPVersion   string             `proxy:"ip-version,omitempty" group:"ip-version,omitempty"`
	DialerProxy proxy.ProxyAdapter `proxy:"dialer-proxy,omitempty"` // don't apply this option into groups, but can set a group name in a proxy
}

// Id implements proxy.ProxyAdapter
func (b *Base) Id() string {
	if b.id == "" {
		b.id = util.NewUUIDV6().String()
	}

	return b.id
}

// MarshalJSON implements proxy.ProxyAdapter
func (b *Base) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"proto": b.Proto().String(),
		"id":    b.Id(),
	})
}

func (b *Base) Addr() string {
	return b.addr
}

func (b *Base) DisableIPv6() bool {
	return b.disableIPv6
}

func (b *Base) Name() string {
	return b.name
}

func (b *Base) Username() string {
	return b.username
}

func (b *Base) Password() string {
	return b.password
}

func (b *Base) Proto() proto.Proto {
	return b.proto
}

// Unwrap implements proxy.Proxy
func (b *Base) Unwrap(metadata *M.Metadata, touch bool) proxy.Proxy {
	return nil
}

// SupportWithDialer implements proxy.ProxyAdapter
func (b *Base) SupportWithDialer() M.Network {
	return M.InvalidNet
}

// SupportUOT implements proxy.ProxyAdapter
func (b *Base) SupportUOT() bool {
	return false
}

// SupportUDP implements proxy.Proxy
func (b *Base) SupportUDP() bool {
	return b.udp
}

// SupportXUDP implements proxy.ProxyAdapter
func (b *Base) SupportXUDP() bool {
	return b.xudp
}

// SupportTFO implements proxy.ProxyAdapter
func (b *Base) SupportTFO() bool {
	return b.tfo
}

// IsL3Protocol implements proxy.Proxy
func (b *Base) IsL3Protocol(metadata *M.Metadata) bool {
	return false
}

// DialContext implements proxy.Proxy
func (b *Base) DialContext(context.Context, *M.Metadata, ...dialer.Option) (P.Conn, error) {
	return nil, E.ErrNotSupport
}

// DialContextWithDialer implements proxy.ProxyAdapter
func (b *Base) DialContextWithDialer(ctx context.Context, dialer P.Dialer, metadata *M.Metadata) (_ P.Conn, err error) {
	return nil, E.ErrNotSupport
}

// ListenPacketContext implements proxy.ProxyAdapter
func (b *Base) ListenPacketContext(context.Context, *M.Metadata, ...dialer.Option) (P.PacketConn, error) {
	return nil, E.ErrNotSupport
}

// ListenPacketWithDialer implements proxy.ProxyAdapter
func (b *Base) ListenPacketWithDialer(ctx context.Context, dialer P.Dialer, metadata *M.Metadata) (_ P.PacketConn, err error) {
	return nil, E.ErrNotSupport
}

// DialOptions return []dialer.Option from struct
func (b *Base) DialOptions(opts ...dialer.Option) []dialer.Option {
	if b.iface != "" {
		opts = append(opts, dialer.WithInterface(b.iface))
	}
	if b.rmark != 0 {
		opts = append(opts, dialer.WithRoutingMark(b.rmark))
	}

	switch b.prefer {
	case C.IPv4Only:
		opts = append(opts, dialer.WithOnlySingleStack(true))
	case C.IPv6Only:
		opts = append(opts, dialer.WithOnlySingleStack(false))
	case C.IPv4Prefer:
		opts = append(opts, dialer.WithPreferIPv4())
	case C.IPv6Prefer:
		opts = append(opts, dialer.WithPreferIPv6())
	default:
	}
	if b.tfo {
		opts = append(opts, dialer.WithTFO(true))
	}

	if b.mpTcp {
		opts = append(opts, dialer.WithMPTCP(true))
	}

	return opts
}
