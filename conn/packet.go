package conn

import (
	"net"
	"time"

	"github.com/lumavpn/luma/common/pool"
	M "github.com/lumavpn/luma/metadata"
)

type PacketReader interface {
	ReadPacket(buffer *pool.Buffer) (destination M.Socksaddr, err error)
}

type PacketWriter interface {
	WritePacket(buffer *pool.Buffer, destination M.Socksaddr) error
}

type PacketConn interface {
	PacketReader
	PacketWriter

	Close() error
	LocalAddr() net.Addr
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}
