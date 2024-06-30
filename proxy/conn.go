package proxy

import (
	"net"
	"strings"
	"syscall"

	"github.com/lumavpn/luma/common/bufio"
	"github.com/lumavpn/luma/conn"
	"github.com/lumavpn/luma/conn/deadline"
	"github.com/lumavpn/luma/proxy/adapter"
)

type Chain = adapter.Chain
type Conn = adapter.Conn
type PacketConn = adapter.PacketConn

type packetConn struct {
	conn.EnhancePacketConn
	chain                   Chain
	adapterName             string
	connID                  string
	actualRemoteDestination string
}

type proxyConn struct {
	conn.ExtendedConn
	chain                   Chain
	actualRemoteDestination string
}

func (c *proxyConn) Chains() Chain {
	return c.chain
}

func (c *proxyConn) AppendToChains(a adapter.ProxyAdapter) {
	c.chain = append(c.chain, a.Name())
}

func (c *proxyConn) RemoteDestination() string {
	return c.actualRemoteDestination
}

func NewConn(c net.Conn, a Proxy) adapter.Conn {
	if _, ok := c.(syscall.Conn); !ok {
		c = conn.NewDeadlineConn(c)
	}
	return &proxyConn{bufio.NewExtendedConn(c), []string{a.Name()},
		parseRemoteDestination(a.Addr())}
}

func (c *packetConn) RemoteDestination() string {
	return c.actualRemoteDestination
}

func (c *packetConn) Chains() Chain {
	return c.chain
}

func (c *packetConn) AppendToChains(a adapter.ProxyAdapter) {
	c.chain = append(c.chain, a.Name())
}

func newPacketConn(pc net.PacketConn, a Proxy) adapter.PacketConn {
	epc := conn.NewEnhancePacketConn(pc)
	if _, ok := pc.(syscall.Conn); !ok {
		epc = deadline.NewEnhancePacketConn(epc)
	}
	return &packetConn{epc, []string{a.Name()}, a.Name(), "", parseRemoteDestination(a.Addr())}
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
