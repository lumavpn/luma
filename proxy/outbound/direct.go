package outbound

import (
	"context"
	"errors"
	"net/netip"

	"github.com/lumavpn/luma/component/loopback"
	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/dns"
	"github.com/lumavpn/luma/dns/resolver"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/proto"
)

type Direct struct {
	*Base
	loopBack *loopback.Detector
}

func NewDirect() *Direct {
	proto := proto.Proto_DIRECT
	return &Direct{
		Base: &Base{
			name:   proto.String(),
			proto:  proto,
			udp:    true,
			prefer: dns.DualStack,
		},
		loopBack: loopback.NewDetector(),
	}
}

func (d *Direct) DialContext(ctx context.Context, metadata *metadata.Metadata, opts ...dialer.Option) (proxy.Conn, error) {
	if err := d.loopBack.CheckConn(metadata); err != nil {
		return nil, err
	}
	opts = append(opts, dialer.WithResolver(resolver.DefaultResolver))
	c, err := dialer.DialContext(ctx, "tcp", metadata.DestinationAddress(), d.Base.DialOptions(opts...)...)
	if err != nil {
		return nil, err
	}
	setKeepAlive(c)
	return d.loopBack.NewConn(NewConn(c, d)), nil
}

func (d *Direct) ListenPacketContext(ctx context.Context, metadata *metadata.Metadata,
	opts ...dialer.Option) (proxy.PacketConn, error) {
	if err := d.loopBack.CheckPacketConn(metadata); err != nil {
		return nil, err
	}

	if !metadata.Resolved() {
		ip, err := resolver.ResolveIPWithResolver(ctx, metadata.Host, resolver.DefaultResolver)
		if err != nil {
			return nil, errors.New("can't resolve ip")
		}
		metadata.DstIP = ip
	}
	pc, err := dialer.NewDialer(d.Base.DialOptions(opts...)...).ListenPacket(ctx, "udp", "",
		netip.AddrPortFrom(metadata.DstIP, metadata.DstPort))
	if err != nil {
		return nil, err
	}
	return d.loopBack.NewPacketConn(newPacketConn(pc, d)), nil

}
