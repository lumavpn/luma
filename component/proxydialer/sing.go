package proxydialer

import (
	"context"
	"net"

	M "github.com/lumavpn/luma/common/metadata"
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/proxy"
)

type SingDialer interface {
	N.Dialer
	SetDialer(dialer proxy.Dialer)
}

type singDialer proxyDialer

var _ N.Dialer = (*singDialer)(nil)

func (d *singDialer) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	return (*proxyDialer)(d).DialContext(ctx, network, destination.String())
}

func (d *singDialer) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	return (*proxyDialer)(d).ListenPacket(ctx, "udp", "", destination.AddrPort())
}

func (d *singDialer) SetDialer(dialer proxy.Dialer) {
	(*proxyDialer)(d).dialer = dialer
}

func NewSingDialer(proxy proxy.ProxyAdapter, dialer proxy.Dialer, statistic bool) SingDialer {
	return (*singDialer)(&proxyDialer{
		proxy:     proxy,
		dialer:    dialer,
		statistic: statistic,
	})
}

type byNameSingDialer struct {
	dialer proxy.Dialer
	proxy  proxy.ProxyAdapter
}

var _ N.Dialer = (*byNameSingDialer)(nil)

func (d *byNameSingDialer) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	return d.dialer.DialContext(ctx, network, destination.String())
}

func (d *byNameSingDialer) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	return d.dialer.ListenPacket(ctx, "udp", "", destination.AddrPort())
}

func (d *byNameSingDialer) SetDialer(dialer proxy.Dialer) {
	d.dialer = dialer
}

func NewProxySingDialer(proxyAdapter proxy.ProxyAdapter, dialer proxy.Dialer) SingDialer {
	var cDialer proxy.Dialer = dialer
	if proxyAdapter != nil {
		cDialer = New(proxyAdapter, dialer, true)
	}
	return &byNameSingDialer{
		dialer: cDialer,
		proxy:  proxyAdapter,
	}
}
