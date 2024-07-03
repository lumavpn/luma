package outbound

import (
	"bytes"
	"context"
	"net"
	"net/netip"
	"time"

	C "github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/component/resolver"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/transport/socks5"
)

const (
	tcpKeepAlivePeriod = 30 * time.Second
)

// setKeepAlive sets tcp keepalive option for tcp connection.
func setKeepAlive(c net.Conn) {
	if tcp, ok := c.(*net.TCPConn); ok {
		tcp.SetKeepAlive(true)
		tcp.SetKeepAlivePeriod(tcpKeepAlivePeriod)
	}
}

// safeConnClose closes tcp connection safely.
func safeConnClose(c net.Conn, err error) {
	if c != nil && err != nil {
		c.Close()
	}
}

func resolveUDPAddr(ctx context.Context, network, address string) (*net.UDPAddr, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	ip, err := resolver.ResolveProxyServerHost(ctx, host)
	if err != nil {
		return nil, err
	}
	return net.ResolveUDPAddr(network, net.JoinHostPort(ip.String(), port))
}

func resolveUDPAddrWithPrefer(ctx context.Context, network, address string, prefer C.DNSPrefer) (*net.UDPAddr, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	var ip netip.Addr
	var fallback netip.Addr
	switch prefer {
	case C.IPv4Only:
		ip, err = resolver.ResolveIPv4ProxyServerHost(ctx, host)
	case C.IPv6Only:
		ip, err = resolver.ResolveIPv6ProxyServerHost(ctx, host)
	case C.IPv6Prefer:
		var ips []netip.Addr
		ips, err = resolver.LookupIPProxyServerHost(ctx, host)
		if err == nil {
			for _, addr := range ips {
				if addr.Is6() {
					ip = addr
					break
				} else {
					if !fallback.IsValid() {
						fallback = addr
					}
				}
			}
		}
	default:
		// C.IPv4Prefer, C.DualStack and other
		var ips []netip.Addr
		ips, err = resolver.LookupIPProxyServerHost(ctx, host)
		if err == nil {
			for _, addr := range ips {
				if addr.Is4() {
					ip = addr
					break
				} else {
					if !fallback.IsValid() {
						fallback = addr
					}
				}
			}

		}
	}

	if !ip.IsValid() && fallback.IsValid() {
		ip = fallback
	}

	if err != nil {
		return nil, err
	}
	return net.ResolveUDPAddr(network, net.JoinHostPort(ip.String(), port))
}

func serializesSocksAddr(metadata *M.Metadata) []byte {
	var buf [][]byte
	addrType := metadata.AddrType()
	aType := uint8(addrType)
	p := uint(metadata.DstPort)
	port := []byte{uint8(p >> 8), uint8(p & 0xff)}
	switch addrType {
	case socks5.AtypDomainName:
		lenM := uint8(len(metadata.Host))
		host := []byte(metadata.Host)
		buf = [][]byte{{aType, lenM}, host, port}
	case socks5.AtypIPv4:
		host := metadata.DstIP.AsSlice()
		buf = [][]byte{{aType}, host, port}
	case socks5.AtypIPv6:
		host := metadata.DstIP.AsSlice()
		buf = [][]byte{{aType}, host, port}
	}
	return bytes.Join(buf, nil)
}
