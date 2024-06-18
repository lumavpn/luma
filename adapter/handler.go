package adapter

import (
	"context"
	"net"

	M "github.com/lumavpn/luma/metadata"
)

type TransportHandler interface {
	HandleTCPConn(conn net.Conn, metadata *M.Metadata)
	HandleUDPPacket(packet UDPPacket, metadata *M.Metadata)
}

type TCPConnectionHandler interface {
	NewConnection(ctx context.Context, conn net.Conn, metadata M.Metadata) error
}

type UDPConnectionHandler interface {
	NewPacketConnection(ctx context.Context, conn net.PacketConn, metadata M.Metadata) error
}
