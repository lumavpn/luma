package callback

import (
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/pool"
	C "github.com/lumavpn/luma/proxy"
)

type firstWriteCallBackConn struct {
	C.Conn
	callback func(error)
	written  bool
}

func (c *firstWriteCallBackConn) Write(b []byte) (n int, err error) {
	defer func() {
		if !c.written {
			c.written = true
			c.callback(err)
		}
	}()
	return c.Conn.Write(b)
}

func (c *firstWriteCallBackConn) WriteBuffer(buffer *pool.Buffer) (err error) {
	defer func() {
		if !c.written {
			c.written = true
			c.callback(err)
		}
	}()
	return c.Conn.WriteBuffer(buffer)
}

func (c *firstWriteCallBackConn) Upstream() any {
	return c.Conn
}

func (c *firstWriteCallBackConn) WriterReplaceable() bool {
	return c.written
}

func (c *firstWriteCallBackConn) ReaderReplaceable() bool {
	return true
}

var _ N.ExtendedConn = (*firstWriteCallBackConn)(nil)

func NewFirstWriteCallBackConn(c C.Conn, callback func(error)) C.Conn {
	return &firstWriteCallBackConn{
		Conn:     c,
		callback: callback,
		written:  false,
	}
}