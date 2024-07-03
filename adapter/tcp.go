package adapter

import (
	"net"

	"github.com/gofrs/uuid/v5"
	N "github.com/lumavpn/luma/common/net"
	M "github.com/lumavpn/luma/metadata"
)

// TCPConn implements the ConnContext and net.Conn interfaces.
type TCPConn interface {
	ConnContext
	Conn() *N.BufferedConn
	Metadata() *M.Metadata
}

func NewTCPConn(c net.Conn, m *M.Metadata) TCPConn {
	id, _ := uuid.NewV4()
	return &tcpConn{
		conn: N.NewBufferedConn(c),
		id:   id,
		m:    m,
	}
}

type tcpConn struct {
	conn *N.BufferedConn
	id   uuid.UUID
	m    *M.Metadata
}

func (c *tcpConn) ID() uuid.UUID {
	return c.id
}

func (c *tcpConn) Conn() *N.BufferedConn {
	return c.conn
}

func (c *tcpConn) Metadata() *M.Metadata {
	return c.m
}
