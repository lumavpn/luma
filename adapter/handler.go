package adapter

// TransportHandler is a TCP/UDP connection handler
type TransportHandler interface {
	HandleTCP(TCPConn)
	HandleUDP(UDPConn)
}
