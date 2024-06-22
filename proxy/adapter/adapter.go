package adapter

import (
	"context"

	"github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/common/atomic"
	"github.com/lumavpn/luma/common/queue"
	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	"github.com/puzpuzpuz/xsync/v3"
)

const (
	defaultHistoriesNum = 10
)

type internalProxyState struct {
	alive   atomic.Bool
	history *queue.Queue[proxy.DelayHistory]
}

type Proxy struct {
	proxy.ProxyAdapter
	alive   atomic.Bool
	history *queue.Queue[proxy.DelayHistory]
	extra   *xsync.MapOf[string, *internalProxyState]
}

func (p *Proxy) AliveForTestUrl(url string) bool {
	if state, ok := p.extra.Load(url); ok {
		return state.alive.Load()
	}

	return p.alive.Load()
}

func (p *Proxy) Dial(m *metadata.Metadata) (proxy.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.DefaultTCPTimeout)
	defer cancel()
	return p.DialContext(ctx, m)
}

func (p *Proxy) DialContext(ctx context.Context, m *metadata.Metadata, opts ...dialer.Option) (proxy.Conn, error) {
	conn, err := p.ProxyAdapter.DialContext(ctx, m, opts...)
	return conn, err
}

func (p *Proxy) DialUDP(m *metadata.Metadata) (proxy.PacketConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.DefaultUDPTimeout)
	defer cancel()
	return p.ListenPacketContext(ctx, m)
}

func NewProxy(adapter proxy.ProxyAdapter) *Proxy {
	return &Proxy{
		ProxyAdapter: adapter,
		history:      queue.New[proxy.DelayHistory](defaultHistoriesNum),
		alive:        atomic.NewBool(true),
		extra:        xsync.NewMapOf[string, *internalProxyState]()}
}
