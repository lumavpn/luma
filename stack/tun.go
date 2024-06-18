package stack

import (
	"context"
	"io"
	"net"
	"net/netip"

	M "github.com/lumavpn/luma/common/metadata"
)

type TCPConnectionHandler interface {
	NewConnection(ctx context.Context, conn net.Conn, metadata M.Metadata) error
}

type UDPConnectionHandler interface {
	NewPacketConnection(ctx context.Context, conn net.PacketConn, metadata M.Metadata) error
}

type Handler interface {
	TCPConnectionHandler
	UDPConnectionHandler
}

type Tun interface {
	io.ReadWriter
	Close() error
}

type Options struct {
	FileDescriptor int
	Name           string
	Inet4Address   []netip.Prefix
	Inet6Address   []netip.Prefix
	MTU            uint32
}
