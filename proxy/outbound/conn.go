package outbound

import (
	"net"
	"strings"
	"syscall"

	"github.com/lumavpn/luma/common/bufio"
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/proxy"
)

type conn struct {
	N.ExtendedConn
	chain                   proxy.Chain
	actualRemoteDestination string
}

func (c *conn) RemoteDestination() string {
	return c.actualRemoteDestination
}

func (c *conn) Chains() proxy.Chain {
	return c.chain
}

func (c *conn) AppendToChains(a proxy.ProxyAdapter) {
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

func NewConn(c net.Conn, a proxy.ProxyAdapter) proxy.Conn {
	if _, ok := c.(syscall.Conn); !ok {
		// c = N.NewDeadlineConn(c) // most conn from outbound can't handle readDeadline correctly
	}
	return &conn{bufio.NewExtendedConn(c), []string{a.Name()}, parseRemoteDestination(a.Addr())}
}

type packetConn struct {
	N.EnhancePacketConn
	chain                   proxy.Chain
	adapterName             string
	connID                  string
	actualRemoteDestination string
}

func (c *packetConn) RemoteDestination() string {
	return c.actualRemoteDestination
}

// Chains implements C.Connection
func (c *packetConn) Chains() proxy.Chain {
	return c.chain
}

// AppendToChains implements C.Connection
func (c *packetConn) AppendToChains(a proxy.ProxyAdapter) {
	c.chain = append(c.chain, a.Name())
}

func (c *packetConn) LocalAddr() net.Addr {
	//lAddr := c.EnhancePacketConn.LocalAddr()
	return nil
	//return N.NewCustomAddr(c.adapterName, c.connID, lAddr)
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

func newPacketConn(pc net.PacketConn, a proxy.ProxyAdapter) proxy.PacketConn {
	epc := N.NewEnhancePacketConn(pc)
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
