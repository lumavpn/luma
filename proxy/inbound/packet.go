package inbound

import (
	"github.com/lumavpn/luma/adapter"
	C "github.com/lumavpn/luma/adapter"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/transport/socks5"
)

// NewPacket is PacketAdapter generator
func NewPacket(target socks5.Addr, packet C.UDPPacket, source proto.Proto, additions ...Option) (adapter.UDPPacket, *M.Metadata) {
	metadata := parseSocksAddr(target)
	metadata.Network = M.UDP
	metadata.Proto = source
	metadata.RawSrcAddr = packet.LocalAddr()
	metadata.RawDstAddr = metadata.UDPAddr()
	WithOptions(metadata, WithSrcAddr(packet.LocalAddr()))
	if p, ok := packet.(C.UDPPacketInAddr); ok {
		WithOptions(metadata, WithInAddr(p.InAddr()))
	}
	WithOptions(metadata, additions...)

	return packet, metadata
}
