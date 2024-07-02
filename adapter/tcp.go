package adapter

import (
	"net"

	"github.com/gofrs/uuid/v5"
	"github.com/lumavpn/luma/conn"
	"github.com/lumavpn/luma/metadata"
)

// TCPConn implements the ConnContext and net.Conn interfaces.
type TCPConn interface {
	ConnContext
	Conn() *conn.BufferedConn
}

func NewTCPConn(c net.Conn, m *metadata.Metadata) TCPConn {
	id, _ := uuid.NewV4()
	return &tcpConn{
		conn: conn.NewBuffConn(c),
		id:   id,
		m:    m,
	}
}

type tcpConn struct {
	conn *conn.BufferedConn
	id   uuid.UUID
	m    *metadata.Metadata
}

func (c *tcpConn) ID() uuid.UUID {
	return c.id
}

func (c *tcpConn) Conn() *conn.BufferedConn {
	return c.conn
}

func (c *tcpConn) Metadata() *metadata.Metadata {
	return c.m
}
