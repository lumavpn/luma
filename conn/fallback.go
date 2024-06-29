package conn

import (
	"net"

	"github.com/lumavpn/luma/common/network"
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/pool"
	M "github.com/lumavpn/luma/metadata"
)

type FallbackPacketConn struct {
	network.PacketConn
	writer network.NetPacketWriter
}

func NewNetPacketConn(conn N.PacketConn) N.NetPacketConn {
	if packetConn, loaded := conn.(N.NetPacketConn); loaded {
		return packetConn
	}
	return &FallbackPacketConn{
		PacketConn: conn,
		writer:     NewNetPacketWriter(conn),
	}
}

func (c *FallbackPacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	buffer := pool.With(p)
	destination, err := c.ReadPacket(buffer)
	if err != nil {
		return
	}
	n = buffer.Len()
	if buffer.Start() > 0 {
		copy(p, buffer.Bytes())
	}
	addr = destination.UDPAddr()
	return
}

func (c *FallbackPacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	return c.writer.WriteTo(p, addr)
}

func (c *FallbackPacketConn) ReaderReplaceable() bool {
	return true
}

func (c *FallbackPacketConn) WriterReplaceable() bool {
	return true
}

func (c *FallbackPacketConn) Upstream() any {
	return c.PacketConn
}

func (c *FallbackPacketConn) UpstreamWriter() any {
	return c.writer
}

type FallbackPacketWriter struct {
	N.PacketWriter
	frontHeadroom int
	rearHeadroom  int
}

func NewNetPacketWriter(writer N.PacketWriter) N.NetPacketWriter {
	if packetWriter, loaded := writer.(N.NetPacketWriter); loaded {
		return packetWriter
	}
	return &FallbackPacketWriter{
		PacketWriter:  writer,
		frontHeadroom: N.CalculateFrontHeadroom(writer),
		rearHeadroom:  N.CalculateRearHeadroom(writer),
	}
}

func (c *FallbackPacketWriter) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if c.frontHeadroom > 0 || c.rearHeadroom > 0 {
		buffer := pool.NewSize(len(p) + c.frontHeadroom + c.rearHeadroom)
		buffer.Resize(c.frontHeadroom, 0)
		buffer.Write(p)
		err = c.PacketWriter.WritePacket(buffer, M.ParseSocksAddrFromNet(addr))
	} else {
		err = c.PacketWriter.WritePacket(pool.As(p), M.ParseSocksAddrFromNet(addr))
	}
	if err != nil {
		return
	}
	n = len(p)
	return
}

func (c *FallbackPacketWriter) WriterReplaceable() bool {
	return true
}

func (c *FallbackPacketWriter) Upstream() any {
	return c.PacketWriter
}
