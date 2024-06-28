package proxy

import (
	"context"
	"errors"
	"net"

	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/proto"
)

type Base struct {
	addr  string
	name  string
	udp   bool
	proto proto.Proto
}

// Addr returns the address of the proxy
func (b *Base) Addr() string {
	return b.addr
}

// Name returns the name of the proxy
func (b *Base) Name() string {
	return b.name
}

// Proto returns the protocol of the proxy
func (b *Base) Proto() proto.Proto {
	return b.proto
}

// SupportUDP returns whether or not the proxy supports UDP
func (b *Base) SupportUDP() bool {
	return b.udp
}

func (b *Base) DialContext(context.Context, *metadata.Metadata) (net.Conn, error) {
	return nil, errors.ErrUnsupported
}

func (b *Base) DialUDP(*metadata.Metadata) (net.PacketConn, error) {
	return nil, errors.ErrUnsupported
}

func (b *Base) ListenPacketContext(context.Context, *metadata.Metadata) (PacketConn, error) {
	return nil, errors.ErrUnsupported
}
