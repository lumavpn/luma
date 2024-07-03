package outbound

import (
	"context"
	"errors"
	"net/netip"

	C "github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/component/loopback"
	"github.com/lumavpn/luma/component/resolver"
	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/proto"
)

type Direct struct {
	*Base
	loopBack *loopback.Detector
}

type DirectOpts struct {
	BasicOption
	Name string `proxy:"name"`
}

// NewDirect creates a new direct dialer that bypasses proxying traffic
func NewDirect() *Direct {
	proto := proto.Proto_Direct
	return &Direct{
		Base: &Base{
			name:   proto.String(),
			proto:  proto,
			udp:    true,
			prefer: C.DualStack,
		},
		loopBack: loopback.NewDetector(),
	}
}

func NewCompatible() *Direct {
	return &Direct{
		Base: &Base{
			name:   "COMPATIBLE",
			proto:  proto.Proto_Compatible,
			udp:    true,
			prefer: C.DualStack,
		},
		loopBack: loopback.NewDetector(),
	}
}

func NewDirectWithOptions(opts DirectOpts) *Direct {
	return &Direct{
		Base: &Base{
			name:   opts.Name,
			proto:  proto.Proto_Direct,
			udp:    true,
			tfo:    opts.TFO,
			mpTcp:  opts.MPTCP,
			iface:  opts.Interface,
			rmark:  opts.RoutingMark,
			prefer: C.NewDNSPrefer(opts.IPVersion),
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
