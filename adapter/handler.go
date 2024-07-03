package adapter

import (
	"net"

	M "github.com/lumavpn/luma/metadata"
)

type TransportHandler interface {
	HandleTCPConn(net.Conn, *M.Metadata)
	HandleUDPPacket(UDPPacket, *M.Metadata)
}
