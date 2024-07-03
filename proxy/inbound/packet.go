package inbound

import (
	C "github.com/lumavpn/luma/adapter"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/transport/socks5"
)

// NewPacket is PacketAdapter generator
func NewPacket(target socks5.Addr, packet C.UDPPacket, source proto.Proto, additions ...Addition) (C.UDPPacket, *M.Metadata) {
	metadata := parseSocksAddr(target)
	metadata.Network = M.UDP
	metadata.Type = source
	metadata.RawSrcAddr = packet.LocalAddr()
	metadata.RawDstAddr = metadata.UDPAddr()
	ApplyAdditions(metadata, WithSrcAddr(packet.LocalAddr()))
	if p, ok := packet.(C.UDPPacketInAddr); ok {
		ApplyAdditions(metadata, WithInAddr(p.InAddr()))
	}
	ApplyAdditions(metadata, additions...)

	return packet, metadata
}
