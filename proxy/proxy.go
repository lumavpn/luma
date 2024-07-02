package proxy

import (
	"context"

	"github.com/lumavpn/luma/dialer"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/proto"
)

type Dialer interface {
	DialContext(context.Context, *M.Metadata, ...dialer.Option) (Conn, error)
	ListenPacketContext(context.Context, *M.Metadata, ...dialer.Option) (PacketConn, error)
}

type ProxyAdapter interface {
	// Name returns the name of this proxy
	Name() string
	// Addr is the address of the proxy
	Addr() string
	// Proto is the protocol of the proxy
	Proto() proto.Proto
	// SupportUDP returns whether or not the proxy supports UDP
	SupportUDP() bool
}

type Proxy interface {
	ProxyAdapter
	URLTest(ctx context.Context, url string) error
}
