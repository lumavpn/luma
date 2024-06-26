package adapter

import (
	"net"

	"github.com/gofrs/uuid/v5"
)

// ConnContext is the default interface to adapt connections
type ConnContext interface {
	ID() uuid.UUID
}

// TCPConn implements the ConnContext and net.Conn interfaces.
type TCPConn interface {
	ConnContext
	net.Conn
}

// UDPConn implements the ConnContext, net.Conn, and net.PacketConn interfaces.
type UDPConn interface {
	ConnContext
	net.Conn
	net.PacketConn
}
