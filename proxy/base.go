package proxy

import (
	"context"
	"errors"
	"net"

	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/dns"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/adapter"
	"github.com/lumavpn/luma/proxy/proto"
)

type Base struct {
	addr     string
	name     string
	udp      bool
	xudp     bool
	tfo      bool
	proto    proto.Proto
	username string
	password string

	iface  string
	rmark  int
	mpTcp  bool
	prefer dns.DNSPrefer
}

/*type BaseOption struct {
	Name     string `proxy:"name,omitempty" group:"name,omitempty"`
	Addr     string `proxy:"addr,omitempty" group:"addr,omitempty"`
	Proto    proto.Proto
	UDP      bool   `proxy:"udp,omitempty" group:"udp,omitempty"`
	Username string `proxy:"username,omitempty" group:"username,omitempty"`
	Password string `proxy:"password,omitempty" group:"password,omitempty"`
	XUDP     bool
	TFO      bool `proxy:"tfo,omitempty" group:"tfo,omitempty"`
	MPTCP    bool

	Interface   string `proxy:"interface-name,omitempty" group:"interface-name,omitempty"`
	RoutingMark int    `proxy:"routing-mark,omitempty" group:"routing-mark,omitempty"`
	IPVersion   string `proxy:"ip-version,omitempty" group:"ip-version,omitempty"`

	Prefer dns.DNSPrefer
}*/

type BasicOption struct {
	Name        string `proxy:"name,omitempty" group:"name,omitempty"`
	Addr        string `proxy:"addr,omitempty" group:"addr,omitempty"`
	UDP         bool   `proxy:"udp,omitempty" group:"udp,omitempty"`
	Username    string `proxy:"username,omitempty" group:"username,omitempty"`
	Password    string `proxy:"password,omitempty" group:"password,omitempty"`
	XUDP        bool   `proxy:"xudp,omitempty" group:"xudp,omitempty"`
	TFO         bool   `proxy:"tfo,omitempty" group:"tfo,omitempty"`
	MPTCP       bool   `proxy:"mptcp,omitempty" group:"mptcp,omitempty"`
	Interface   string `proxy:"interface-name,omitempty" group:"interface-name,omitempty"`
	RoutingMark int    `proxy:"routing-mark,omitempty" group:"routing-mark,omitempty"`
	IPVersion   string `proxy:"ip-version,omitempty" group:"ip-version,omitempty"`
}

/*func NewBase(opts *BaseOption) *Base {
	return &Base{
		name:     opts.Name,
		addr:     opts.Addr,
		username: opts.Username,
		password: opts.Password,
		udp:      opts.UDP,
		xudp:     opts.XUDP,
		tfo:      opts.TFO,
		mpTcp:    opts.MPTCP,
		iface:    opts.Interface,
		rmark:    opts.RoutingMark,
		prefer:   opts.Prefer,
	}
}*/

// Addr returns the address of the proxy
func (b *Base) Addr() string {
	return b.addr
}

// Name returns the name of the proxy
func (b *Base) Name() string {
	return b.name
}

// Proto returns the protocol of the proxy
func (b *Base) Proto() proto.Proto {
	return b.proto
}

// SupportUDP returns whether or not the proxy supports UDP
func (b *Base) SupportUDP() bool {
	return b.udp
}

func (b *Base) DialContext(context.Context, *metadata.Metadata, ...dialer.Option) (net.Conn, error) {
	return nil, errors.ErrUnsupported
}

func (b *Base) ListenPacketContext(context.Context, *metadata.Metadata, ...dialer.Option) (adapter.PacketConn, error) {
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

	switch b.prefer {
	case dns.IPv4Only:
		opts = append(opts, dialer.WithOnlySingleStack(true))
	case dns.IPv6Only:
		opts = append(opts, dialer.WithOnlySingleStack(false))
	case dns.IPv4Prefer:
		opts = append(opts, dialer.WithPreferIPv4())
	case dns.IPv6Prefer:
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
