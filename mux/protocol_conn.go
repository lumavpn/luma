package mux

import (
	"net"

	"github.com/lumavpn/luma/common/bufio"
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/pool"
)

type protocolConn struct {
	net.Conn
	request        Request
	requestWritten bool
}

func newProtocolConn(conn net.Conn, request Request) net.Conn {
	writer, isVectorised := bufio.CreateVectorisedWriter(conn)
	if isVectorised {
		return &vectorisedProtocolConn{
			protocolConn{
				Conn:    conn,
				request: request,
			},
			writer,
		}
	} else {
		return &protocolConn{
			Conn:    conn,
			request: request,
		}
	}
}

func (c *protocolConn) NeedHandshake() bool {
	return !c.requestWritten
}

func (c *protocolConn) Write(p []byte) (n int, err error) {
	if c.requestWritten {
		return c.Conn.Write(p)
	}
	buffer := EncodeRequest(c.request, p)
	n, err = c.Conn.Write(buffer.Bytes())
	buffer.Release()
	if err == nil {
		n--
	}
	c.requestWritten = true
	return n, err
}

func (c *protocolConn) Upstream() any {
	return c.Conn
}

type vectorisedProtocolConn struct {
	protocolConn
	writer N.VectorisedWriter
}

func (c *vectorisedProtocolConn) WriteVectorised(buffers []*pool.Buffer) error {
	if c.requestWritten {
		return c.writer.WriteVectorised(buffers)
	}
	c.requestWritten = true
	buffer := EncodeRequest(c.request, nil)
	return c.writer.WriteVectorised(append([]*pool.Buffer{buffer}, buffers...))
}
