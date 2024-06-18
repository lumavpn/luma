package util

import (
	"net"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var KeepAliveInterval = 15 * time.Second

func TCPKeepAlive(c net.Conn) {
	if tcp, ok := c.(*net.TCPConn); ok {
		_ = tcp.SetKeepAlive(true)
		_ = tcp.SetKeepAlivePeriod(KeepAliveInterval)
	}
}

func CalculateInterfaceName(name string) string {
	var tunName string
	if runtime.GOOS == "darwin" {
		tunName = "utun"
	} else if name != "" {
		return name
	} else {
		tunName = "tun"
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return tunName
	}
	var tunIndex int
	for _, netInterface := range interfaces {
		if strings.HasPrefix(netInterface.Name, tunName) {
			index, parseErr := strconv.ParseInt(netInterface.Name[len(tunName):], 10, 16)
			if parseErr == nil {
				tunIndex = int(index) + 1
			}
		}
	}
	return strconv.FormatInt(int64(tunIndex), 10)
}
