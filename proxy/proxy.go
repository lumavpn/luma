package proxy

import (
	"context"
	"net"
	"net/netip"
	"time"

	"github.com/lumavpn/luma/proxy/adapter"
)

const (
	tcpConnectTimeout = 5 * time.Second
)

type Proxy interface {
	adapter.ProxyAdapter
}

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
	ListenPacket(ctx context.Context, network, address string, rAddrPort netip.AddrPort) (net.PacketConn, error)
}
