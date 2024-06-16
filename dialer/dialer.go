package dialer

import (
	"context"
	"net"
	"syscall"

	"go.uber.org/atomic"
)

var (
	DefaultInterfaceName  = atomic.NewString("")
	DefaultInterfaceIndex = atomic.NewInt32(0)
	DefaultRoutingMark    = atomic.NewInt32(0)
)

func DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return DialContextWithOptions(ctx, network, address,
		WithInterface(DefaultInterfaceName.Load()),
		WithInterfaceIndex(int(DefaultInterfaceIndex.Load())),
		WithRoutingMark(int(DefaultRoutingMark.Load())),
	)
}

func DialContextWithOptions(ctx context.Context, network, address string, options ...Option) (net.Conn, error) {
	d := &net.Dialer{
		Control: func(network, address string, c syscall.RawConn) error {
			if len(options) == 0 {
				return nil
			}
			opts := &option{}
			for _, op := range options {
				op(opts)
			}
			return setSocketOptions(network, address, c, opts)
		},
	}
	return d.DialContext(ctx, network, address)
}
