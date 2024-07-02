package adapter

import (
	M "github.com/lumavpn/luma/metadata"
)

type TransportHandler interface {
	HandleTCPConn(TCPConn)
	HandleUDPPacket(packet UDPPacket, metadata *M.Metadata)
}
