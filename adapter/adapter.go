package adapter

import (
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

// ConnContext is the default interface to adapt connections
type ConnContext interface {
	ID() *stack.TransportEndpointID
}

// TCPConn implements the ConnContext and net.Conn interfaces.
type TCPConn interface {
	Conn() *gonet.TCPConn
	ConnContext
}

// UDPConn implements the ConnContext, net.Conn, and net.PacketConn interfaces.
type UDPConn interface {
	ConnContext
	Conn() *gonet.UDPConn
}

func NewTCPConn(conn *gonet.TCPConn, id stack.TransportEndpointID) TCPConn {
	return &tcpConn{
		conn: conn,
		id:   id,
	}
}

func NewUDPConn(conn *gonet.UDPConn, id stack.TransportEndpointID) UDPConn {
	return &udpConn{
		conn: conn,
		id:   id,
	}
}

type tcpConn struct {
	conn *gonet.TCPConn
	id   stack.TransportEndpointID
}

func (c *tcpConn) Conn() *gonet.TCPConn {
	return c.conn
}

func (c *tcpConn) ID() *stack.TransportEndpointID {
	return &c.id
}

type udpConn struct {
	conn *gonet.UDPConn
	id   stack.TransportEndpointID
}

func (c *udpConn) Conn() *gonet.UDPConn {
	return c.conn
}

func (c *udpConn) ID() *stack.TransportEndpointID {
	return &c.id
}
