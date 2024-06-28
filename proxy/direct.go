package proxy

import (
	"context"
	"net"

	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/proto"
)

type Direct struct {
	*Base
}

func NewDirect() *Direct {
	return &Direct{
		Base: &Base{
			proto: proto.Proto_DIRECT,
		},
	}
}

func (d *Direct) DialContext(ctx context.Context, metadata *metadata.Metadata) (net.Conn, error) {
	c, err := dialer.DialContext(ctx, "tcp", metadata.DestinationAddress())
	if err != nil {
		return nil, err
	}
	setKeepAlive(c)
	return c, nil
}

func (d *Direct) ListenPacketContext(ctx context.Context, m *metadata.Metadata) (PacketConn, error) {
	pc, err := dialer.ListenPacket("udp", "")
	if err != nil {
		return nil, err
	}
	return newPacketConn(pc, d), nil
}
