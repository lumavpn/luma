package adapter

import (
	"net"

	"github.com/gofrs/uuid/v5"
	"github.com/lumavpn/luma/conn"
	M "github.com/lumavpn/luma/metadata"
)

type ConnContext interface {
	ID() uuid.UUID
}

type TCPConn interface {
	ConnContext
	Metadata() *M.Metadata
	Conn() *conn.BuffConn
}

type UDPConn interface {
	ConnContext
	Metadata() *M.Metadata
	PacketConn() net.PacketConn
}
