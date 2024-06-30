package loopback

import (
	"sync"

	"github.com/lumavpn/luma/common/pool"
	"github.com/lumavpn/luma/conn"
	"github.com/lumavpn/luma/proxy/adapter"
)

type firstWriteCallBackConn struct {
	adapter.Conn
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

func NewFirstWriteCallBackConn(c adapter.Conn, callback func(error)) adapter.Conn {
	return &firstWriteCallBackConn{
		Conn:     c,
		callback: callback,
		written:  false,
	}
}

type closeCallbackConn struct {
	adapter.Conn
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

func NewCloseCallbackConn(conn adapter.Conn, callback func()) adapter.Conn {
	return &closeCallbackConn{Conn: conn, closeFunc: callback}
}

type closeCallbackPacketConn struct {
	adapter.PacketConn
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

func NewCloseCallbackPacketConn(conn adapter.PacketConn, callback func()) adapter.PacketConn {
	return &closeCallbackPacketConn{PacketConn: conn, closeFunc: callback}
}
