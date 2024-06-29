package callback

import (
	"sync"

	"github.com/lumavpn/luma/common/pool"
	"github.com/lumavpn/luma/conn"
	"github.com/lumavpn/luma/proxy"
)

type firstWriteCallBackConn struct {
	proxy.Conn
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

var _ conn.ExtendedConn = (*firstWriteCallBackConn)(nil)

func NewFirstWriteCallBackConn(c proxy.Conn, callback func(error)) proxy.Conn {
	return &firstWriteCallBackConn{
		Conn:     c,
		callback: callback,
		written:  false,
	}
}

type closeCallbackConn struct {
	proxy.Conn
	closeFunc func()
	closeOnce sync.Once
}

func (w *closeCallbackConn) Close() error {
	w.closeOnce.Do(w.closeFunc)
	return w.Conn.Close()
}

func (w *closeCallbackConn) ReaderReplaceable() bool {
	return true
}

func (w *closeCallbackConn) WriterReplaceable() bool {
	return true
}

func (w *closeCallbackConn) Upstream() any {
	return w.Conn
}

func NewCloseCallbackConn(conn proxy.Conn, callback func()) proxy.Conn {
	return &closeCallbackConn{Conn: conn, closeFunc: callback}
}

type closeCallbackPacketConn struct {
	proxy.PacketConn
	closeFunc func()
	closeOnce sync.Once
}

func (w *closeCallbackPacketConn) Close() error {
	w.closeOnce.Do(w.closeFunc)
	return w.PacketConn.Close()
}

func (w *closeCallbackPacketConn) ReaderReplaceable() bool {
	return true
}

func (w *closeCallbackPacketConn) WriterReplaceable() bool {
	return true
}

func (w *closeCallbackPacketConn) Upstream() any {
	return w.PacketConn
}

func NewCloseCallbackPacketConn(conn proxy.PacketConn, callback func()) proxy.PacketConn {
	return &closeCallbackPacketConn{PacketConn: conn, closeFunc: callback}
}
