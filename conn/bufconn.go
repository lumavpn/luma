// Package bufconn provides a net.Conn implemented by a buffer
package conn

import (
	"bufio"
	"net"
)

// BufConn provides a net.Conn implemented by a buffer
type BufConn struct {
	r *bufio.Reader
	net.Conn
	peeked bool
}

func NewBufConn(c net.Conn) *BufConn {
	if bc, ok := c.(*BufConn); ok {
		return bc
	}
	return &BufConn{bufio.NewReader(c), c, false}
}

// Reader returns the internal bufio.Reader
func (c *BufConn) Reader() *bufio.Reader {
	return c.r
}

func (c *BufConn) ResetPeeked() {
	c.peeked = false
}

func (c *BufConn) Peeked() bool {
	return c.peeked
}

// Peek returns the next n bytes without advancing the reader
func (c *BufConn) Peek(n int) ([]byte, error) {
	c.peeked = true
	return c.r.Peek(n)
}

func (c *BufConn) Discard(n int) (discarded int, err error) {
	return c.r.Discard(n)
}

func (c *BufConn) Read(p []byte) (int, error) {
	return c.r.Read(p)
}

func (c *BufConn) ReadByte() (byte, error) {
	return c.r.ReadByte()
}

func (c *BufConn) UnreadByte() error {
	return c.r.UnreadByte()
}

func (c *BufConn) Buffered() int {
	return c.r.Buffered()
}
