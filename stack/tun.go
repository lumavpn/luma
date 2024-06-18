package stack

import (
	"io"
	"net/netip"

	"github.com/lumavpn/luma/adapter"
)

type Handler interface {
	adapter.TCPConnectionHandler
	adapter.UDPConnectionHandler
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
