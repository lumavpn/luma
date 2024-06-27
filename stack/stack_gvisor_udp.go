//go:build with_gvisor

package stack

import (
	"context"

	"github.com/lumavpn/luma/adapter"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
)

func (t *gVisor) withUDPHandler(ctx context.Context, ipStack *stack.Stack) func(r *udp.ForwarderRequest) {
	return func(r *udp.ForwarderRequest) {
		var (
			wq waiter.Queue
			id = r.ID()
		)
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

		conn := adapter.NewUDPConn(udpConn, id)

		// go t.Handler.NewPacketConnection
		go t.handler.NewPacketConnection(ctx, conn)
	}
}
