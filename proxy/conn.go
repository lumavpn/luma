package proxy

import (
	"github.com/lumavpn/luma/common/network"
)

type Connection interface {
	Chains() Chain
	AppendToChains(adapter ProxyAdapter)
	RemoteDestination() string
}

type Conn interface {
	network.ExtendedConn
	Connection
}

type PacketConn interface {
	network.EnhancePacketConn
	Connection
}
