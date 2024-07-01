package proxy

import (
	"context"
	"net"
	"net/netip"
	"time"

	"github.com/lumavpn/luma/dialer"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/proto"
)

const (
	tcpConnectTimeout = 5 * time.Second
)

type ProxyAdapter interface {
	// Name returns the name of this proxy
	Name() string
	// Addr is the address of the proxy
	Addr() string
	// Proto is the protocol of the proxy
	Proto() proto.Proto
	// SupportUDP returns whether or not the proxy supports UDP
	SupportUDP() bool

	DialContext(context.Context, *M.Metadata, ...dialer.Option) (Conn, error)
	ListenPacketContext(context.Context, *M.Metadata, ...dialer.Option) (PacketConn, error)

	Unwrap(metadata *M.Metadata, touch bool) ProxyAdapter
}

type Proxy interface {
	ProxyAdapter
}

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
	ListenPacket(ctx context.Context, network, address string, rAddrPort netip.AddrPort) (net.PacketConn, error)
}
