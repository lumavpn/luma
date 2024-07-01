package proxy

import (
	"net"
	"strings"
	"syscall"

	"github.com/lumavpn/luma/common/bufio"
	"github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/network/deadline"
	cconn "github.com/lumavpn/luma/conn"
	"github.com/lumavpn/luma/proxy/adapter"
)

type Chain = adapter.Chain
type Conn = adapter.Conn
type PacketConn = adapter.PacketConn

type conn struct {
	network.ExtendedConn
	chain                   Chain
	actualRemoteDestination string
}

func (c *conn) RemoteDestination() string {
	return c.actualRemoteDestination
}

func (c *conn) Chains() Chain {
	return c.chain
}

func (c *conn) AppendToChains(a adapter.ProxyAdapter) {
	c.chain = append(c.chain, a.Name())
}

func (c *conn) Upstream() any {
	return c.ExtendedConn
}

func (c *conn) WriterReplaceable() bool {
	return true
}

func (c *conn) ReaderReplaceable() bool {
	return true
}

func NewConn(c net.Conn, a Proxy) adapter.Conn {
	if _, ok := c.(syscall.Conn); !ok {
		c = cconn.NewDeadlineConn(c)
	}
	return &conn{bufio.NewExtendedConn(c), []string{a.Name()},
		parseRemoteDestination(a.Addr())}
}

type packetConn struct {
	network.EnhancePacketConn
	chain                   Chain
	adapterName             string
	connID                  string
	actualRemoteDestination string
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

func (c *packetConn) LocalAddr() net.Addr {
	lAddr := c.EnhancePacketConn.LocalAddr()
	return network.NewCustomAddr(c.adapterName, c.connID, lAddr)
}

func (c *packetConn) Upstream() any {
	return c.EnhancePacketConn
}

func (c *packetConn) WriterReplaceable() bool {
	return true
}

func (c *packetConn) ReaderReplaceable() bool {
	return true
}

func newPacketConn(pc net.PacketConn, a Proxy) adapter.PacketConn {
	epc := network.NewEnhancePacketConn(pc)
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
