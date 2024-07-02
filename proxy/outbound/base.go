package outbound

import (
	"context"
	"errors"
	"net"

	"github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/dialer"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
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
	prefer common.DNSPrefer
}

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

// IsL3Protocol implements proxy.Proxy
func (b *Base) IsL3Protocol(metadata *M.Metadata) bool {
	return false
}

func (b *Base) DialContext(context.Context, *M.Metadata, ...dialer.Option) (net.Conn, error) {
	return nil, errors.ErrUnsupported
}

func (b *Base) ListenPacketContext(context.Context, *M.Metadata, ...dialer.Option) (proxy.PacketConn, error) {
	return nil, errors.ErrUnsupported
}

func (b *Base) Unwrap(metadata *M.Metadata, touch bool) proxy.ProxyAdapter {
	return nil
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
	case common.IPv4Only:
		opts = append(opts, dialer.WithOnlySingleStack(true))
	case common.IPv6Only:
		opts = append(opts, dialer.WithOnlySingleStack(false))
	case common.IPv4Prefer:
		opts = append(opts, dialer.WithPreferIPv4())
	case common.IPv6Prefer:
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
