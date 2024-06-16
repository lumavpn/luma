package adapter

import (
	"net"

	"github.com/gofrs/uuid/v5"
	M "github.com/lumavpn/luma/metadata"
)

type ConnContext interface {
	ID() uuid.UUID
}

type TCPConn interface {
	ConnContext
	Metadata() *M.Metadata
	Conn() net.Conn
}

type UDPConn interface {
	ConnContext
	Metadata() *M.Metadata
	PacketConn() net.PacketConn
}
