package metadata

import (
	"net"
	"net/netip"
	"strconv"
	"unsafe"
)

type Socksaddr struct {
	Addr netip.Addr
	Port uint16
	Fqdn string
}

func SocksaddrFromNetIP(ap netip.AddrPort) Socksaddr {
	return Socksaddr{
		Addr: ap.Addr(),
		Port: ap.Port(),
	}
}

func (ap Socksaddr) IsIP() bool {
	return ap.Addr.IsValid()
}

func (ap Socksaddr) IsIPv4() bool {
	return ap.Addr.Is4()
}

func (ap Socksaddr) IsIPv6() bool {
	return ap.Addr.Is6()
}

func (ap Socksaddr) Unwrap() Socksaddr {
	if ap.Addr.Is4In6() {
		return Socksaddr{
			Addr: netip.AddrFrom4(ap.Addr.As4()),
			Port: ap.Port,
		}
	}
	return ap
}

func (ap Socksaddr) IsFqdn() bool {
	return IsDomainName(ap.Fqdn)
}

func (ap Socksaddr) IsValid() bool {
	return ap.IsIP() || ap.IsFqdn()
}

func (ap Socksaddr) Network() string {
	return "socks"
}

func (ap Socksaddr) AddrString() string {
	if ap.Addr.IsValid() {
		return ap.Addr.String()
	} else {
		return ap.Fqdn
	}
}

func (ap Socksaddr) IPAddr() *net.IPAddr {
	return &net.IPAddr{
		IP:   ap.Addr.AsSlice(),
		Zone: ap.Addr.Zone(),
	}
}

func (ap Socksaddr) TCPAddr() *net.TCPAddr {
	return &net.TCPAddr{
		IP:   ap.Addr.AsSlice(),
		Port: int(ap.Port),
		Zone: ap.Addr.Zone(),
	}
}

func (ap Socksaddr) UDPAddr() *net.UDPAddr {
	return &net.UDPAddr{
		IP:   ap.Addr.AsSlice(),
		Port: int(ap.Port),
		Zone: ap.Addr.Zone(),
	}
}

func (ap Socksaddr) AddrPort() netip.AddrPort {
	return *(*netip.AddrPort)(unsafe.Pointer(&ap))
}

func (ap Socksaddr) String() string {
	return net.JoinHostPort(ap.AddrString(), strconv.Itoa(int(ap.Port)))
}

func parseAddrFromIP(ip net.IP) netip.Addr {
	addr, _ := netip.AddrFromSlice(ip)
	return addr
}

func parseAddrPortFromNet(netAddr net.Addr) netip.AddrPort {
	var ip net.IP
	var port uint16
	switch addr := netAddr.(type) {
	case Socksaddr:
		return addr.AddrPort()
	case *net.TCPAddr:
		ip = addr.IP
		port = uint16(addr.Port)
	case *net.UDPAddr:
		ip = addr.IP
		port = uint16(addr.Port)
	case *net.IPAddr:
		ip = addr.IP
	}
	return netip.AddrPortFrom(parseAddrFromIP(ip), port)
}

func unwrapIPv6Address(address string) string {
	if len(address) > 2 && address[0] == '[' && address[len(address)-1] == ']' {
		return address[1 : len(address)-1]
	}
	return address
}

func parseSocksAddr(address string) Socksaddr {
	host, portStr, _ := net.SplitHostPort(address)
	port, _ := strconv.Atoi(portStr)
	netAddr, err := netip.ParseAddr(unwrapIPv6Address(host))
	if err != nil {
		return Socksaddr{
			Fqdn: host,
			Port: uint16(port),
		}
	} else {
		return Socksaddr{
			Addr: netAddr,
			Port: uint16(port),
		}
	}
}

func SocksaddrFrom(addr netip.Addr, port uint16) Socksaddr {
	return SocksaddrFromNetIP(netip.AddrPortFrom(addr, port))
}

func ParseSocksAddrFromNet(ap net.Addr) Socksaddr {
	if ap == nil {
		return Socksaddr{}
	}
	if socksAddr, ok := ap.(Socksaddr); ok {
		return socksAddr
	}
	addr := SocksaddrFromNetIP(parseAddrPortFromNet(ap))
	if addr.IsValid() {
		return addr
	}
	return parseSocksAddr(ap.String())
}

func ParseAddrFromNet(netAddr net.Addr) netip.Addr {
	if addr := parseAddrPortFromNet(netAddr); addr.Addr().IsValid() {
		return addr.Addr()
	}
	switch addr := netAddr.(type) {
	case Socksaddr:
		return addr.Addr
	case *net.IPAddr:
		return parseAddrFromIP(addr.IP)
	case *net.IPNet:
		return parseAddrFromIP(addr.IP)
	default:
		return netip.Addr{}
	}
}
