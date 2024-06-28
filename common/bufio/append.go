package bufio

import (
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/pool"
	"github.com/lumavpn/luma/util"
)

type appendConn struct {
	N.ExtendedConn
	reader N.ExtendedReader
	writer N.ExtendedWriter
}

func NewAppendConn(conn N.ExtendedConn, reader N.ExtendedReader, writer N.ExtendedWriter) N.ExtendedConn {
	return &appendConn{
		ExtendedConn: conn,
		reader:       reader,
		writer:       writer,
	}
}

func (c *appendConn) Read(p []byte) (n int, err error) {
	if c.reader == nil {
		return c.ExtendedConn.Read(p)
	} else {
		return c.reader.Read(p)
	}
}

func (c *appendConn) ReadBuffer(buffer *pool.Buffer) error {
	if c.reader == nil {
		return c.ExtendedConn.ReadBuffer(buffer)
	} else {
		return c.reader.ReadBuffer(buffer)
	}
}

func (c *appendConn) Write(p []byte) (n int, err error) {
	if c.writer == nil {
		return c.ExtendedConn.Write(p)
	} else {
		return c.writer.Write(p)
	}
}

func (c *appendConn) WriteBuffer(buffer *pool.Buffer) error {
	if c.writer == nil {
		return c.ExtendedConn.WriteBuffer(buffer)
	} else {
		return c.writer.WriteBuffer(buffer)
	}
}

func (c *appendConn) Close() error {
	return util.Close(
		c.ExtendedConn,
		c.reader,
		c.writer,
	)
}

func (c *appendConn) UpstreamReader() any {
	return c.reader
}

func (c *appendConn) ReaderReplaceable() bool {
	return true
}

func (c *appendConn) UpstreamWriter() any {
	return c.writer
}

func (c *appendConn) WriterReplaceable() bool {
	return true
}

func (c *appendConn) Upstream() any {
	return c.ExtendedConn
}
