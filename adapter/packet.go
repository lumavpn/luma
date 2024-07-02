package adapter

import (
	"net"

	M "github.com/lumavpn/luma/metadata"
)

type WriteBack interface {
	WriteBack(b []byte, addr net.Addr) (n int, err error)
}

// UDPPacket contains the data of UDP packet, and offers control/info of UDP packet's source
type UDPPacket interface {
	// Data get the payload of UDP Packet
	Data() []byte

	// WriteBack writes the payload with source IP/Port equals addr
	WriteBack

	// Drop call after packet is used, could recycle buffer in this function.
	Drop()

	// LocalAddr returns the source IP/Port of packet
	LocalAddr() net.Addr
}

type UDPPacketInAddr interface {
	InAddr() net.Addr
}

// PacketAdapter is a UDP Packet adapter for socks/redir/tun
type PacketAdapter interface {
	UDPPacket
	Metadata() *M.Metadata
}

type packetAdapter struct {
	UDPPacket
	metadata *M.Metadata
}

// Metadata returns destination metadata
func (s *packetAdapter) Metadata() *M.Metadata {
	return s.metadata
}

func NewPacketAdapter(packet UDPPacket, metadata *M.Metadata) PacketAdapter {
	return &packetAdapter{
		packet,
		metadata,
	}
}
