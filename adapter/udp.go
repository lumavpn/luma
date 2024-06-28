package adapter

import (
	"github.com/gofrs/uuid/v5"
	"github.com/lumavpn/luma/conn"
	"github.com/lumavpn/luma/metadata"
)

// UDPConn implements the ConnContext, net.Conn, and net.PacketConn interfaces.
type UDPConn interface {
	ConnContext
	Conn() conn.PacketConn
}

func NewUDPConn(conn conn.PacketConn, m *metadata.Metadata) UDPConn {
	id, _ := uuid.NewV4()
	return &udpConn{
		conn: conn,
		id:   id,
		m:    m,
	}
}

type udpConn struct {
	conn conn.PacketConn
	id   uuid.UUID
	m    *metadata.Metadata
}

func (c *udpConn) ID() uuid.UUID {
	return c.id
}

func (c *udpConn) Conn() conn.PacketConn {
	return c.conn
}

func (c *udpConn) Metadata() *metadata.Metadata {
	return c.m
}
