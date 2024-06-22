package dialer

import (
	"context"
	"net"
	"net/netip"
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

func ListenPacket(ctx context.Context, network, address string, rAddrPort netip.AddrPort, options ...Option) (net.PacketConn, error) {
	return nil, nil
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

func ParseNetwork(network string, addr netip.Addr) string {
	return network
}

type Dialer struct {
	Opt option
}

func (d Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return DialContext(ctx, network, address, WithOption(d.Opt))
}

func (d Dialer) ListenPacket(ctx context.Context, network, address string, rAddrPort netip.AddrPort) (net.PacketConn, error) {
	opt := WithOption(d.Opt)
	if rAddrPort.Addr().Unmap().IsLoopback() {
		// avoid "The requested address is not valid in its context."
		opt = WithInterface("")
	}
	return ListenPacket(ctx, ParseNetwork(network, rAddrPort.Addr()), address, rAddrPort, opt)
}

func NewDialer(options ...Option) Dialer {
	opt := applyOptions(options...)
	return Dialer{Opt: *opt}
}
