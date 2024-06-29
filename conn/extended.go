package conn

import (
	"io"
	"net"

	"github.com/lumavpn/luma/common/pool"
)

type ExtendedReader interface {
	io.Reader
	ReadBuffer(buffer *pool.Buffer) error
}

type ExtendedWriter interface {
	io.Writer
	WriteBuffer(buffer *pool.Buffer) error
}

type ExtendedConn interface {
	ExtendedReader
	ExtendedWriter
	net.Conn
}
