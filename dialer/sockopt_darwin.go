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

		if opts.interfaceIndex == 0 && opts.interfaceName != "" {
			if iface, err := net.InterfaceByName(opts.interfaceName); err == nil {
				opts.interfaceIndex = iface.Index
			}
		}

		if opts.interfaceIndex != 0 {
			switch network {
			case "tcp4", "udp4":
				innerErr = unix.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_BOUND_IF, opts.interfaceIndex)
			case "tcp6", "udp6":
				innerErr = unix.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, syscall.IPV6_BOUND_IF, opts.interfaceIndex)
			}
			if innerErr != nil {
				return
			}
		}
	})

	if innerErr != nil {
		err = innerErr
	}
	return
}
