//go:build with_gvisor

package stack

import (
	"context"
	"net"

	"github.com/lumavpn/luma/adapter"
	M "github.com/lumavpn/luma/common/metadata"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/inbound"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/waiter"
)

type gTCPConn struct {
	*gonet.TCPConn
}

func (c *gTCPConn) Upstream() any {
	return c.TCPConn
}

func (c *gTCPConn) Write(b []byte) (n int, err error) {
	n, err = c.TCPConn.Write(b)
	if err == nil {
		return
	}
	err = wrapError(err)
	return
}

func (t *gVisor) withTCPHandler(ctx context.Context, ipStack *stack.Stack) func(r *tcp.ForwarderRequest) {
	return func(r *tcp.ForwarderRequest) {
		var wq waiter.Queue
		endpoint, err := r.CreateEndpoint(&wq)
		if err != nil {
			r.Complete(true)
			return
		}
		r.Complete(false)

		err = setSocketOptions(ipStack, endpoint)

		tcpConn := gonet.NewTCPConn(&wq, endpoint)
		lAddr := tcpConn.RemoteAddr()
		rAddr := tcpConn.LocalAddr()
		if lAddr == nil || rAddr == nil {
			tcpConn.Close()
			return
		}

		go func() {
			m := &metadata.Metadata{
				Network:     metadata.TCP,
				Source:      M.ParseSocksAddrFromNet(lAddr),
				Destination: M.ParseSocksAddrFromNet(rAddr),
			}
			inbound.WithOptions(m, inbound.WithDstAddr(m.Destination), inbound.WithSrcAddr(m.Source),
				inbound.WithLocalAddr(tcpConn.LocalAddr()))

			hErr := t.handler.NewConnection(ctx, adapter.NewTCPConn(&gTCPConn{tcpConn}, m))
			if hErr != nil {
				endpoint.Abort()
			}
		}()
	}
}

func wrapError(err error) error {
	if opErr, isOpErr := err.(*net.OpError); isOpErr {
		switch opErr.Err.Error() {
		case "endpoint is closed for send",
			"endpoint is closed for receive",
			"operation aborted":
			return net.ErrClosed
		}
	}
	return err
}
