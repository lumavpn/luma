package proxy

import (
	"net"
	"strings"
	"syscall"

	"github.com/lumavpn/luma/conn"
)

type Connection interface {
	Chains() Chain
	AppendToChains(adapter Proxy)
	RemoteDestination() string
}

type PacketConn interface {
	conn.EnhancePacketConn
	Connection
}

type packetConn struct {
	conn.EnhancePacketConn
	chain                   Chain
	adapterName             string
	connID                  string
	actualRemoteDestination string
}

func (c *packetConn) RemoteDestination() string {
	return c.actualRemoteDestination
}

// Chains implements C.Connection
func (c *packetConn) Chains() Chain {
	return c.chain
}

// AppendToChains implements C.Connection
func (c *packetConn) AppendToChains(a Proxy) {
	c.chain = append(c.chain, a.Name())
}

func newPacketConn(pc net.PacketConn, a Proxy) PacketConn {
	epc := conn.NewEnhancePacketConn(pc)
	if _, ok := pc.(syscall.Conn); !ok { // exclusion system conn like *net.UDPConn
		//epc = N.NewDeadlineEnhancePacketConn(epc) // most conn from outbound can't handle readDeadline correctly
	}
	return &packetConn{epc, []string{a.Name()}, a.Name(), "", /*util.NewUUIDV4().String()*/
		parseRemoteDestination(a.Addr())}
}

func parseRemoteDestination(addr string) string {
	if dst, _, err := net.SplitHostPort(addr); err == nil {
		return dst
	} else {
		if addrError, ok := err.(*net.AddrError); ok && strings.Contains(addrError.Err, "missing port") {
			return dst
		} else {
			return ""
		}
	}
}
