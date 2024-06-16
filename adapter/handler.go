package adapter

import (
	"net"

	M "github.com/lumavpn/luma/metadata"
)

type TransportHandler interface {
	HandleTCPConn(conn net.Conn, metadata *M.Metadata)
	HandleUDPPacket(packet UDPPacket, metadata *M.Metadata)
}
