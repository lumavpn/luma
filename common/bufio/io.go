package bufio

import (
	"io"
	"net"

	M "github.com/lumavpn/luma/common/metadata"
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/pool"
)

// Deprecated: bad usage
func ReadBuffer(reader N.ExtendedReader, buffer *pool.Buffer) (n int, err error) {
	n, err = reader.Read(buffer.FreeBytes())
	buffer.Truncate(n)
	return
}

// Deprecated: bad usage
func ReadPacket(reader N.PacketReader, buffer *pool.Buffer) (n int, addr net.Addr, err error) {
	startLen := buffer.Len()
	addr, err = reader.ReadPacket(buffer)
	n = buffer.Len() - startLen
	return
}

func Write(writer io.Writer, data []byte) (n int, err error) {
	if extendedWriter, isExtended := writer.(N.ExtendedWriter); isExtended {
		return WriteBuffer(extendedWriter, pool.As(data))
	} else {
		return writer.Write(data)
	}
}

func WriteBuffer(writer N.ExtendedWriter, buffer *pool.Buffer) (n int, err error) {
	frontHeadroom := N.CalculateFrontHeadroom(writer)
	rearHeadroom := N.CalculateRearHeadroom(writer)
	if frontHeadroom > buffer.Start() || rearHeadroom > buffer.FreeLen() {
		newBuffer := pool.NewSize(buffer.Len() + frontHeadroom + rearHeadroom)
		newBuffer.Resize(frontHeadroom, 0)
		newBuffer.Write(buffer.Bytes())
		buffer.Release()
		buffer = newBuffer
	}
	dataLen := buffer.Len()
	err = writer.WriteBuffer(buffer)
	if err == nil {
		n = dataLen
	}
	return
}

func WritePacket(writer N.NetPacketWriter, data []byte, addr net.Addr) (n int, err error) {
	if extendedWriter, isExtended := writer.(N.PacketWriter); isExtended {
		return WritePacketBuffer(extendedWriter, pool.As(data), M.ParseSocksAddrFromNet(addr))
	} else {
		return writer.WriteTo(data, addr)
	}
}

func WritePacketBuffer(writer N.PacketWriter, buffer *pool.Buffer, destination M.Socksaddr) (n int, err error) {
	frontHeadroom := N.CalculateFrontHeadroom(writer)
	rearHeadroom := N.CalculateRearHeadroom(writer)
	if frontHeadroom > buffer.Start() || rearHeadroom > buffer.FreeLen() {
		newBuffer := pool.NewSize(buffer.Len() + frontHeadroom + rearHeadroom)
		newBuffer.Resize(frontHeadroom, 0)
		newBuffer.Write(buffer.Bytes())
		buffer.Release()
		buffer = newBuffer
	}
	dataLen := buffer.Len()
	err = writer.WritePacket(buffer, destination)
	if err == nil {
		n = dataLen
	}
	return
}

func WriteVectorised(writer N.VectorisedWriter, data [][]byte) (n int, err error) {
	var dataLen int
	buffers := make([]*pool.Buffer, 0, len(data))
	for _, p := range data {
		dataLen += len(p)
		buffers = append(buffers, pool.As(p))
	}
	err = writer.WriteVectorised(buffers)
	if err == nil {
		n = dataLen
	}
	return
}

func WriteVectorisedPacket(writer N.VectorisedPacketWriter, data [][]byte, destination M.Socksaddr) (n int, err error) {
	var dataLen int
	buffers := make([]*pool.Buffer, 0, len(data))
	for _, p := range data {
		dataLen += len(p)
		buffers = append(buffers, pool.As(p))
	}
	err = writer.WriteVectorisedPacket(buffers, destination)
	if err == nil {
		n = dataLen
	}
	return
}
