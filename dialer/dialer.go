package dialer

import (
	"context"
	"net"
	"syscall"

	"go.uber.org/atomic"
)

var (
	DefaultOptions        []Option
	DefaultInterfaceName  = atomic.NewString("")
	DefaultInterfaceIndex = atomic.NewInt32(0)
	DefaultRoutingMark    = atomic.NewInt32(0)
)

func DialContext(ctx context.Context, network, address string, options ...Option) (net.Conn, error) {
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

func ListenPacket(network, address string) (net.PacketConn, error) {
	return ListenPacketWithOptions(network, address,
		WithInterface(DefaultInterfaceName.Load()),
		WithInterfaceIndex(int(DefaultInterfaceIndex.Load())),
		WithRoutingMark(int(DefaultRoutingMark.Load())),
	)
}

func ListenPacketWithOptions(network, address string, options ...Option) (net.PacketConn, error) {
	lc := &net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			opts := &option{}
			for _, op := range options {
				op(opts)
			}
			return setSocketOptions(network, address, c, opts)
		},
	}
	return lc.ListenPacket(context.Background(), network, address)
}
