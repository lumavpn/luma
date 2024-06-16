package dialer

import (
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

func setSocketOptions(network, address string, c syscall.RawConn, opts *option) (err error) {
	if opts == nil || !isTCPSocket(network) && !isUDPSocket(network) {
		return
	}

	var innerErr error
	err = c.Control(func(fd uintptr) {
		host, _, _ := net.SplitHostPort(address)
		if ip := net.ParseIP(host); ip != nil && !ip.IsGlobalUnicast() {
			return
		}

		if opts.routingMark != 0 {
			if innerErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_USER_COOKIE, opts.routingMark); innerErr != nil {
				return
			}
		}
	})

	if innerErr != nil {
		err = innerErr
	}
	return
}
