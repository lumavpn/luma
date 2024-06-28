package conn

import "net"

type NetPacketReader interface {
	PacketReader
	ReadFrom(p []byte) (n int, addr net.Addr, err error)
}

type NetPacketWriter interface {
	PacketWriter
	WriteTo(p []byte, addr net.Addr) (n int, err error)
}

type NetPacketConn interface {
	PacketConn
	NetPacketReader
	NetPacketWriter
}
