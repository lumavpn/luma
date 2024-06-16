package inbound

import (
	"github.com/lumavpn/luma/adapter"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/protos"
	"github.com/lumavpn/luma/transport/socks5"
)

func NewPacket(target socks5.Addr, packet adapter.UDPPacket, source protos.Protocol, options ...Option) (adapter.UDPPacket, *M.Metadata) {
	metadata := parseSocksAddr(target)
	metadata.Network = M.UDP
	metadata.Type = source
	metadata.RawSrcAddr = packet.LocalAddr()
	metadata.RawDstAddr = metadata.UDPAddr()
	WithOptions(metadata, WithSrcAddr(packet.LocalAddr()))
	if p, ok := packet.(adapter.UDPPacketInAddr); ok {
		WithOptions(metadata, WithInAddr(p.InAddr()))
	}
	WithOptions(metadata, options...)
	return packet, metadata
}
