//go:build with_gvisor

package stack

import (
	"context"

	"github.com/lumavpn/luma/adapter"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/waiter"
)

func (t *gVisor) withTCPHandler(ctx context.Context, ipStack *stack.Stack) func(r *tcp.ForwarderRequest) {
	return func(r *tcp.ForwarderRequest) {
		var (
			wq waiter.Queue
			id = r.ID()
		)
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
			// go t.Handler.NewConnection
			hErr := t.handler.NewConnection(ctx, adapter.NewTCPConn(tcpConn, id))

			if hErr != nil {
				endpoint.Abort()
			}
		}()
	}
}
