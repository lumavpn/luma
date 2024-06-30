package adapter

import (
	"context"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/dialer"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/proto"
)

type WriteBackProxy interface {
	adapter.WriteBack
	UpdateWriteBack(wb adapter.WriteBack)
}

type ProxyAdapter interface {
	// Name returns the name of this proxy
	Name() string
	// Addr is the address of the proxy
	Addr() string
	// Proto is the protocol of the proxy
	Proto() proto.Proto
	// SupportUDP returns whether or not the proxy supports UDP
	SupportUDP() bool

	DialContext(context.Context, *M.Metadata, ...dialer.Option) (Conn, error)
	ListenPacketContext(context.Context, *M.Metadata, ...dialer.Option) (PacketConn, error)

	Unwrap(metadata *M.Metadata, touch bool) ProxyAdapter
}

type Proxy struct {
	ProxyAdapter
}

func NewProxy(proxy ProxyAdapter) *Proxy {
	return &Proxy{proxy}
}

// DialContext implements C.ProxyAdapter
func (p *Proxy) DialContext(ctx context.Context, metadata *M.Metadata, opts ...dialer.Option) (Conn, error) {
	conn, err := p.ProxyAdapter.DialContext(ctx, metadata, opts...)
	return conn, err
}
