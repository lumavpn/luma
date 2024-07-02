package adapter

import (
	"net"

	"github.com/gofrs/uuid/v5"
)

// ConnContext is the default interface to adapt connections
type ConnContext interface {
	ID() uuid.UUID
}

// UDPConn implements the ConnContext, net.Conn, and net.PacketConn interfaces.
type UDPConn interface {
	ConnContext
	net.Conn
	net.PacketConn
}
