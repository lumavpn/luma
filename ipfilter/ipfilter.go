package ipfilter

import (
	"net"
	"net/netip"

	M "github.com/lumavpn/luma/metadata"
)

var (
	lanAllowedIPs    []netip.Prefix
	lanDisAllowedIPs []netip.Prefix
)

func SetAllowedIPs(allowedIPs []netip.Prefix) {
	lanAllowedIPs = allowedIPs
}

func SetDisAllowedIPs(disAllowedIPs []netip.Prefix) {
	lanDisAllowedIPs = disAllowedIPs
}

func AllowedIPs() []netip.Prefix {
	return lanAllowedIPs
}

func DisAllowedIPs() []netip.Prefix {
	return lanDisAllowedIPs
}

func IsRemoteAddrDisAllowed(addr net.Addr) bool {
	m := M.Metadata{}
	if err := m.SetRemoteAddr(addr); err != nil {
		return false
	}
	return isAllowed(m.AddrPort().Addr().Unmap()) && !isDisAllowed(m.AddrPort().Addr().Unmap())
}

func isAllowed(addr netip.Addr) bool {
	if addr.IsValid() {
		for _, prefix := range lanAllowedIPs {
			if prefix.Contains(addr) {
				return true
			}
		}
	}
	return false
}

func isDisAllowed(addr netip.Addr) bool {
	if addr.IsValid() {
		for _, prefix := range lanDisAllowedIPs {
			if prefix.Contains(addr) {
				return true
			}
		}
	}
	return false
}
