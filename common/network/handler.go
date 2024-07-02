package network

import (
	"context"
	"net"

	M "github.com/lumavpn/luma/metadata"
)

type TCPConnectionHandler interface {
	NewConnection(ctx context.Context, conn net.Conn, metadata M.Metadata) error
}

type UDPConnectionHandler interface {
	NewPacketConnection(ctx context.Context, conn PacketConn, metadata M.Metadata) error
}
