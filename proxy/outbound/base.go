package outbound

import (
	"context"
	"encoding/json"
	"errors"

	C "github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/dialer"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/proto"
)

type Base struct {
	name string
	// addr is the address used to connect to the proxy
	addr string
	// proto is the protocol of the proxy
	proto proto.Proto
	// udp indicates whether or not the proxy supports UDP
	udp         bool
	xudp        bool
	tfo         bool
	disableIPv6 bool
	username    string
	password    string
	rmark       int
	mpTcp       bool
	id          string
	iface       string
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

type BasicOption struct {
	TFO         bool               `proxy:"tfo,omitempty" group:"tfo,omitempty"`
	MPTCP       bool               `proxy:"mptcp,omitempty" group:"mptcp,omitempty"`
	Interface   string             `proxy:"interface-name,omitempty" group:"interface-name,omitempty"`
	RoutingMark int                `proxy:"routing-mark,omitempty" group:"routing-mark,omitempty"`
	IPVersion   string             `proxy:"ip-version,omitempty" group:"ip-version,omitempty"`
	DialerProxy proxy.ProxyAdapter `proxy:"dialer-proxy,omitempty"`
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

func (b *Base) Addr() string {
	return b.addr
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

// MarshalJSON implements proxy.ProxyAdapter
func (b *Base) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"proto": b.Proto().String(),
		"addr":  b.Addr(),
	})
}

// Unwrap implements proxy.Proxy
func (b *Base) Unwrap(metadata *M.Metadata, touch bool) proxy.Proxy {
	return nil
}

// SupportWithDialer implements proxy.ProxyAdapter
func (b *Base) SupportWithDialer() M.Network {
	return M.InvalidNet
}

// IsL3Protocol implements proxy.Proxy
func (b *Base) IsL3Protocol(metadata *M.Metadata) bool {
	return false
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

// DialContext implements proxy.Proxy
func (b *Base) DialContext(context.Context, *M.Metadata, ...dialer.Option) (proxy.Conn, error) {
	return nil, errors.ErrUnsupported
}

// DialContextWithDialer implements proxy.ProxyAdapter
func (b *Base) DialContextWithDialer(ctx context.Context, dialer proxy.Dialer, metadata *M.Metadata) (_ proxy.Conn, err error) {
	return nil, errors.ErrUnsupported
}

// ListenPacketContext implements proxy.ProxyAdapter
func (b *Base) ListenPacketContext(context.Context, *M.Metadata, ...dialer.Option) (proxy.PacketConn, error) {
	return nil, errors.ErrUnsupported
}

// ListenPacketWithDialer implements proxy.ProxyAdapter
func (b *Base) ListenPacketWithDialer(ctx context.Context, dialer proxy.Dialer, metadata *M.Metadata) (_ proxy.PacketConn, err error) {
	return nil, errors.ErrUnsupported
}

// DialOptions return []dialer.Option from struct
func (b *Base) DialOptions(opts ...dialer.Option) []dialer.Option {
	if b.iface != "" {
		opts = append(opts, dialer.WithInterface(b.iface))
	}

	if b.rmark != 0 {
		opts = append(opts, dialer.WithRoutingMark(b.rmark))
	}

	return opts
}
