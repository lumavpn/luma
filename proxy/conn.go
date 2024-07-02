package proxy

import "net"

type Connection interface {
	Chains() Chain
	AppendToChains(adapter ProxyAdapter)
}

type Conn interface {
	net.Conn
	Connection
}

type PacketConn interface {
	net.PacketConn
	Connection
}
