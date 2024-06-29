package proxy

import (
	"context"
	"net"
	"net/netip"
	"time"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/proto"
)

const (
	tcpConnectTimeout = 5 * time.Second
)

type Proxy interface {
	// Name returns the name of this proxy
	Name() string
	// Addr is the address of the proxy
	Addr() string
	// Proto is the protocol of the proxy
	Proto() proto.Proto
	// SupportUDP returns whether or not the proxy supports UDP
	SupportUDP() bool

	DialContext(context.Context, *metadata.Metadata, ...dialer.Option) (Conn, error)
	ListenPacketContext(context.Context, *metadata.Metadata, ...dialer.Option) (PacketConn, error)
}

type WriteBackProxy interface {
	adapter.WriteBack
	UpdateWriteBack(wb adapter.WriteBack)
}

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
	ListenPacket(ctx context.Context, network, address string, rAddrPort netip.AddrPort) (net.PacketConn, error)
}
