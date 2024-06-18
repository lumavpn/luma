package util

import (
	"net"
	"time"
)

var KeepAliveInterval = 15 * time.Second

func TCPKeepAlive(c net.Conn) {
	if tcp, ok := c.(*net.TCPConn); ok {
		_ = tcp.SetKeepAlive(true)
		_ = tcp.SetKeepAlivePeriod(KeepAliveInterval)
	}
}
