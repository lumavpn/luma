package outbound

import (
	"context"
	"net/netip"

	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/protos"
)

type Direct struct {
	*Base
}

// NewDirect returns a new instance of a direct outbound proxy
func NewDirect() *Direct {
	at := protos.AdapterType_Direct
	return &Direct{
		Base: &Base{
			name: at.String(),
			at:   at,
			udp:  true,
		},
	}
}

// NewDirectWithOptions returns a new instance of Direct configured with the given options
func NewDirectWithOptions(opts BaseOptions) *Direct {
	return &Direct{
		Base: &Base{
			interfaceName: opts.Interface,
			routingMark:   opts.RoutingMark,
			name:          opts.Name,
			at:            protos.AdapterType_Direct,
			udp:           true,
		},
	}
}

// DialContext connects to the address on the network using the provided Metadata
func (d *Direct) DialContext(ctx context.Context, m *metadata.Metadata, opts ...dialer.Option) (proxy.Conn, error) {
	c, err := dialer.DialContext(ctx, "tcp", m.DestinationAddress())
	if err != nil {
		return nil, err
	}
	setKeepAlive(c)
	return NewConn(c, d), nil
}

func (d *Direct) ListenPacketContext(ctx context.Context, m *metadata.Metadata, opts ...dialer.Option) (proxy.PacketConn, error) {
	pc, err := dialer.NewDialer(d.Base.DialOptions(opts...)...).ListenPacket(ctx, "udp", "",
		netip.AddrPortFrom(m.DstIP, m.DstPort))
	if err != nil {
		return nil, err
	}
	return newPacketConn(pc, d), nil
}
