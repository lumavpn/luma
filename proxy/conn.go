package proxy

import (
	N "github.com/lumavpn/luma/common/network"
)

type Connection interface {
	Chains() Chain
	AppendToChains(adapter ProxyAdapter)
	RemoteDestination() string
}

type Conn interface {
	N.ExtendedConn
	Connection
}

type PacketConn interface {
	N.EnhancePacketConn
	Connection
}
