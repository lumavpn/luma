package proxy

import (
	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/proxy/proto"
)

type Base struct {
	name string
	// addr is the address used to connect to the proxy
	addr string
	// proto is the protocol of the proxy
	proto proto.Proto
	// udp indicates whether or not the proxy supports UDP
	udp bool

	iface string
	rmark int
}

func (b *Base) Addr() string {
	return b.addr
}

func (b *Base) Proto() proto.Proto {
	return b.proto
}

func (b *Base) SupportUDP() bool {
	return false
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
