package dns

import (
	"context"
	"net"

	"github.com/lumavpn/luma/dialer"
)

type dnsDialer func(ctx context.Context, network, addr string) (net.Conn, error)

func getDialer(r *Resolver, proxyName string, opts ...dialer.Option) dnsDialer {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		opts = append(opts, dialer.WithResolver(r))
		return dialer.DialContext(ctx, network, addr, opts...)
	}
}
