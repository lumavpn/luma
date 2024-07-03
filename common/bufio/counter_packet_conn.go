package bufio

import (
	"sync/atomic"

	"github.com/lumavpn/luma/common/buf"
	M "github.com/lumavpn/luma/common/metadata"
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/util"
)

type CounterPacketConn struct {
	N.PacketConn
	readCounter  []N.CountFunc
	writeCounter []N.CountFunc
}

func NewInt64CounterPacketConn(conn N.PacketConn, readCounter []*atomic.Int64, writeCounter []*atomic.Int64) *CounterPacketConn {
	return &CounterPacketConn{
		conn,
		util.Map(readCounter, func(it *atomic.Int64) N.CountFunc {
			return func(n int64) {
				it.Add(n)
			}
		}),
		util.Map(writeCounter, func(it *atomic.Int64) N.CountFunc {
			return func(n int64) {
				it.Add(n)
			}
		}),
	}
}

func NewCounterPacketConn(conn N.PacketConn, readCounter []N.CountFunc, writeCounter []N.CountFunc) *CounterPacketConn {
	return &CounterPacketConn{conn, readCounter, writeCounter}
}

func (c *CounterPacketConn) ReadPacket(buffer *buf.Buffer) (destination M.Socksaddr, err error) {
	destination, err = c.PacketConn.ReadPacket(buffer)
	if err == nil {
		if buffer.Len() > 0 {
			for _, counter := range c.readCounter {
				counter(int64(buffer.Len()))
			}
		}
	}
	return
}

func (c *CounterPacketConn) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	dataLen := int64(buffer.Len())
	err := c.PacketConn.WritePacket(buffer, destination)
	if err != nil {
		return err
	}
	if dataLen > 0 {
		for _, counter := range c.writeCounter {
			counter(dataLen)
		}
	}
	return nil
}

func (c *CounterPacketConn) UnwrapPacketReader() (N.PacketReader, []N.CountFunc) {
	return c.PacketConn, c.readCounter
}

func (c *CounterPacketConn) UnwrapPacketWriter() (N.PacketWriter, []N.CountFunc) {
	return c.PacketConn, c.writeCounter
}

func (c *CounterPacketConn) Upstream() any {
	return c.PacketConn
}
