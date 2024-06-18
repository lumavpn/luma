package network

import (
	"net"
	"net/netip"

	M "github.com/lumavpn/luma/common/metadata"
	"github.com/lumavpn/luma/util"
)

func LocalAddrs() ([]netip.Addr, error) {
	interfaceAddrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	return util.Map(interfaceAddrs, M.AddrFromNetAddr), nil
}

func IsPublicAddr(addr netip.Addr) bool {
	return !(addr.IsPrivate() ||
		addr.IsLoopback() ||
		addr.IsMulticast() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsInterfaceLocalMulticast() ||
		addr.IsUnspecified())
}

func IsVirtual(addr netip.Addr) bool {
	return addr.IsLoopback() || addr.IsMulticast() || addr.IsInterfaceLocalMulticast()
}

func LocalPublicAddrs() ([]netip.Addr, error) {
	publicAddrs, err := LocalAddrs()
	if err != nil {
		return nil, err
	}
	return util.Filter(publicAddrs, IsPublicAddr), nil
}
