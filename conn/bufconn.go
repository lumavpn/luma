package conn

import (
	"bufio"
	"net"
)

type BuffConn struct {
	r *bufio.Reader
	net.Conn
	peeked bool
}

func NewBuffConn(c net.Conn) *BuffConn {
	if bc, ok := c.(*BuffConn); ok {
		return bc
	}
	return &BuffConn{bufio.NewReader(c), c, false}
}

// Reader returns the internal bufio.Reader
func (c *BuffConn) Reader() *bufio.Reader {
	return c.r
}

func (c *BuffConn) ResetPeeked() {
	c.peeked = false
}

func (c *BuffConn) Peeked() bool {
	return c.peeked
}

// Peek returns the next n bytes without advancing the reader
func (c *BuffConn) Peek(n int) ([]byte, error) {
	c.peeked = true
	return c.r.Peek(n)
}

func (c *BuffConn) Discard(n int) (discarded int, err error) {
	return c.r.Discard(n)
}
