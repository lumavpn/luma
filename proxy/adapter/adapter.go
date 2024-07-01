package adapter

import (
	"context"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/dialer"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
)

type WriteBackProxy interface {
	adapter.WriteBack
	UpdateWriteBack(wb adapter.WriteBack)
}

type Proxy struct {
	proxy.ProxyAdapter
}

func NewProxy(proxy proxy.ProxyAdapter) *Proxy {
	return &Proxy{proxy}
}

// DialContext implements C.ProxyAdapter
func (p *Proxy) DialContext(ctx context.Context, metadata *M.Metadata, opts ...dialer.Option) (proxy.Conn, error) {
	conn, err := p.ProxyAdapter.DialContext(ctx, metadata, opts...)
	return conn, err
}
