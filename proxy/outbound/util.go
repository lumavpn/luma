package outbound

import (
	"net"
	"time"
)

const (
	tcpKeepAlivePeriod = 30 * time.Second
)

// setKeepAlive sets the TCP keepalive option for a TCP connection
func setKeepAlive(c net.Conn) {
	if tcp, ok := c.(*net.TCPConn); ok {
		tcp.SetKeepAlive(true)
		tcp.SetKeepAlivePeriod(tcpKeepAlivePeriod)
	}
}
