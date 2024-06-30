package adapter

import "github.com/lumavpn/luma/conn"

type Connection interface {
	Chains() Chain
	AppendToChains(adapter ProxyAdapter)
	RemoteDestination() string
}

type Conn interface {
	conn.ExtendedConn
	Connection
}

type PacketConn interface {
	conn.EnhancePacketConn
	Connection
}
