package adapter

import (
	"context"
	"net"

	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/pool"
	M "github.com/lumavpn/luma/metadata"
)

// TransportHandler is a TCP/UDP connection handler
type TransportHandler interface {
	HandleTCP(TCPConn)
	HandleUDP(PacketAdapter)
}

type TCPConnectionHandler interface {
	NewConnection(ctx context.Context, conn net.Conn, metadata M.Metadata) error
}

type UDPHandler interface {
	NewPacket(ctx context.Context, conn N.PacketConn, buffer *pool.Buffer, metadata M.Metadata) error
}

type UDPConnectionHandler interface {
	NewPacketConnection(ctx context.Context, conn N.PacketConn, metadata M.Metadata) error
}
