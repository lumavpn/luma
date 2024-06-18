package stack

import (
	"context"
	"io"
	"net"
	"net/netip"

	"github.com/lumavpn/luma/common/errors"
	M "github.com/lumavpn/luma/common/metadata"
	"github.com/lumavpn/luma/common/network"
)

type TCPConnectionHandler interface {
	NewConnection(ctx context.Context, conn net.Conn, metadata M.Metadata) error
}

type UDPConnectionHandler interface {
	NewPacketConnection(ctx context.Context, conn network.PacketConn, metadata M.Metadata) error
}

type Handler interface {
	TCPConnectionHandler
	UDPConnectionHandler
	errors.ErrorHandler
}

type Tun interface {
	io.ReadWriter
	network.VectorisedWriter
	Close() error
}

type WinTun interface {
	Tun
	ReadPacket() ([]byte, func(), error)
}

type LinuxTUN interface {
	Tun
	network.FrontHeadroom
	BatchSize() int
	BatchRead(buffers [][]byte, offset int, readN []int) (n int, err error)
	BatchWrite(buffers [][]byte, offset int) error
	TXChecksumOffload() bool
}

type Options struct {
	FileDescriptor int
	Name           string
	Inet4Address   []netip.Prefix
	Inet6Address   []netip.Prefix
	MTU            uint32
}
