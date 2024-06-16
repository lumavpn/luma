package adapter

import (
	"net"

	M "github.com/lumavpn/luma/metadata"
)

// UDPPacket contains the data of a UDP packet. It includes the ability to control/access the UDP packet's source
type UDPPacket interface {
	// Data returns the payload of the UDP Packet
	Data() []byte

	// Drop is called after a packet is used and no longer needed
	Drop()

	// LocalAddr returns the source IP/Port of packet
	LocalAddr() net.Addr
}

type UDPPacketInAddr interface {
	InAddr() net.Addr
}

type packetAdapter struct {
	UDPPacket
	metadata *M.Metadata
}

// // PacketAdapter is a UDP Packet adapter
type PacketAdapter interface {
	UDPPacket
	Metadata() *M.Metadata
}

// Metadata returns the destination metadata of the packet
func (s *packetAdapter) Metadata() *M.Metadata {
	return s.metadata
}

// NewPacketAdapter creates a new instance of PacketAdapter
func NewPacketAdapter(packet UDPPacket, metadata *M.Metadata) PacketAdapter {
	return &packetAdapter{
		packet,
		metadata,
	}
}
