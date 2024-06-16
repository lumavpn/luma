package dialer

import (
	"context"
	"net"

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
		WithRoutingMark(DefaultRoutingMark.Load()),
	)
}

func DialContextWithOptions(ctx context.Context, network, address string, options ...Option) (net.Conn, error) {
	d := &net.Dialer{}
	return d.DialContext(ctx, network, address)
}
