package dialer

import (
	"encoding/binary"
	"net"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	IP_UNICAST_IF   = 31
	IPV6_UNICAST_IF = 31
)

func setSocketOptions(network, address string, c syscall.RawConn, opts *option) (err error) {
	if opts == nil || !isTCPSocket(network) && !isUDPSocket(network) {
		return
	}

	var innerErr error
	err = c.Control(func(fd uintptr) {
		host, _, _ := net.SplitHostPort(address)
		ip := net.ParseIP(host)
		if ip != nil && !ip.IsGlobalUnicast() {
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
				innerErr = bindSocketToInterface4(windows.Handle(fd), uint32(opts.interfaceIndex))
			case "tcp6", "udp6":
				innerErr = bindSocketToInterface6(windows.Handle(fd), uint32(opts.interfaceIndex))
				if network == "udp6" && ip == nil {
					// The underlying IP net maybe IPv4 even if the `network` param is `udp6`,
					// so we should bind socket to interface4 at the same time.
					innerErr = bindSocketToInterface4(windows.Handle(fd), uint32(opts.interfaceIndex))
				}
			}
		}
	})

	if innerErr != nil {
		err = innerErr
	}
	return
}

func bindSocketToInterface4(handle windows.Handle, index uint32) error {
	var bytes [4]byte
	binary.BigEndian.PutUint32(bytes[:], index)
	index = *(*uint32)(unsafe.Pointer(&bytes[0]))
	return windows.SetsockoptInt(handle, windows.IPPROTO_IP, IP_UNICAST_IF, int(index))
}

func bindSocketToInterface6(handle windows.Handle, index uint32) error {
	return windows.SetsockoptInt(handle, windows.IPPROTO_IPV6, IPV6_UNICAST_IF, int(index))
}
