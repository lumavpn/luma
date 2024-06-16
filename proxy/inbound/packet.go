package inbound

import (
	"github.com/lumavpn/luma/adapter"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/protos"
	"github.com/lumavpn/luma/transport/socks5"
)

func NewPacket(target socks5.Addr, packet adapter.UDPPacket, source protos.Protocol) (adapter.UDPPacket, *M.Metadata) {
	metadata := parseSocksAddr(target)
	metadata.Network = M.UDP
	metadata.Type = source
	metadata.RawSrcAddr = packet.LocalAddr()
	metadata.RawDstAddr = metadata.UDPAddr()

	return packet, metadata
}
