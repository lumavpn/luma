package proxydialer

import (
	"context"
	"net"

	M "github.com/lumavpn/luma/common/metadata"
	"github.com/lumavpn/luma/component/slowdown"
)

type SlowDownSingDialer struct {
	SingDialer
	Slowdown *slowdown.SlowDown
}

func (d SlowDownSingDialer) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	return slowdown.Do(d.Slowdown, ctx, func() (net.Conn, error) {
		return d.SingDialer.DialContext(ctx, network, destination)
	})
}

func (d SlowDownSingDialer) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	return slowdown.Do(d.Slowdown, ctx, func() (net.PacketConn, error) {
		return d.SingDialer.ListenPacket(ctx, destination)
	})
}

func NewSlowDownSingDialer(d SingDialer, sd *slowdown.SlowDown) SlowDownSingDialer {
	return SlowDownSingDialer{
		SingDialer: d,
		Slowdown:   sd,
	}
}
