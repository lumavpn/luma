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

// contains returns whether or not any of the given networks includes ip.
func contains(networks []netip.Prefix, ip netip.Addr) bool {
	if ip.IsValid() {
		for _, prefix := range networks {
			if prefix.Contains(ip) {
				return true
			}
		}
	}
	return false
}

// isAllowed returns whether or not the list of allowed IPs includes ip.
func isAllowed(ip netip.Addr) bool {
	return contains(lanAllowedIPs, ip)
}

// isDisAllowed returns whether or not the list of disallowed IPs includes ip.
func isDisAllowed(ip netip.Addr) bool {
	return contains(lanDisAllowedIPs, ip)
}
