//go:build with_gvisor

package stack

import (
	"context"
	"time"

	"github.com/lumavpn/luma/common/bufio"
	"github.com/lumavpn/luma/common/canceler"
	M "github.com/lumavpn/luma/common/metadata"

	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
)

func (t *gVisor) withUDPHandler(ctx context.Context, ipStack *stack.Stack) func(r *udp.ForwarderRequest) {
	return func(r *udp.ForwarderRequest) {
		var wq waiter.Queue
		endpoint, err := r.CreateEndpoint(&wq)
		if err != nil {
			return
		}
		udpConn := gonet.NewUDPConn(&wq, endpoint)
		lAddr := udpConn.RemoteAddr()
		rAddr := udpConn.LocalAddr()
		if lAddr == nil || rAddr == nil {
			endpoint.Abort()
			return
		}

		gConn := &gUDPConn{UDPConn: udpConn}

		go func() {
			var m M.Metadata
			m.Source = M.ParseSocksAddrFromNet(lAddr)
			m.Destination = M.ParseSocksAddrFromNet(rAddr)
			ctx, conn := canceler.NewPacketConn(ctx, bufio.NewUnbindPacketConnWithAddr(gConn, m.Destination),
				time.Duration(t.udpTimeout)*time.Second)
			hErr := t.handler.NewPacketConnection(ctx, conn, m)
			if hErr != nil {
				endpoint.Abort()
			}
		}()
	}
}

type gUDPConn struct {
	*gonet.UDPConn
}

func (c *gUDPConn) Read(b []byte) (n int, err error) {
	n, err = c.UDPConn.Read(b)
	if err == nil {
		return
	}
	err = wrapError(err)
	return
}

func (c *gUDPConn) Write(b []byte) (n int, err error) {
	n, err = c.UDPConn.Write(b)
	if err == nil {
		return
	}
	err = wrapError(err)
	return
}

func (c *gUDPConn) Close() error {
	return c.UDPConn.Close()
}
