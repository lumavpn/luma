package bufio

import (
	"io"
	"net"

	M "github.com/lumavpn/luma/metadata"
)

type AddrConn struct {
	net.Conn
	M.Metadata
}

func (c *AddrConn) LocalAddr() net.Addr {
	if c.Metadata.Destination.IsValid() {
		return c.Metadata.Destination.TCPAddr()
	}
	return c.Conn.LocalAddr()
}

func (c *AddrConn) RemoteAddr() net.Addr {
	if c.Metadata.Source.IsValid() {
		return c.Metadata.Source.TCPAddr()
	}
	return c.Conn.RemoteAddr()
}

func (c *AddrConn) ReadFrom(r io.Reader) (n int64, err error) {
	return io.Copy(c.Conn, r)
}

func (c *AddrConn) WriteTo(w io.Writer) (n int64, err error) {
	return io.Copy(w, c.Conn)
}

func (c *AddrConn) ReaderReplaceable() bool {
	return true
}

func (c *AddrConn) WriterReplaceable() bool {
	return true
}

func (c *AddrConn) Upstream() any {
	return c.Conn
}