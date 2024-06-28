package listener

import (
	"net"
	"net/netip"
	"sync"

	"github.com/lumavpn/luma/adapter"
)

type BaseListener struct {
	listener   net.Listener
	addr       string
	closed     bool
	mu         sync.Mutex
	listenAddr netip.Addr
	tunnel     adapter.TransportHandler
}

type BaseOptions struct {
	Addr     string
	Listener net.Listener
	Tunnel   adapter.TransportHandler
}

func New(opts BaseOptions) (*BaseListener, error) {
	if opts.Addr == "" {
		opts.Addr = "0.0.0.0"
	}
	addr, err := netip.ParseAddr(opts.Addr)
	if err != nil {
		return nil, err
	}

	baseListener := &BaseListener{
		listenAddr: addr,
		tunnel:     opts.Tunnel,
	}
	return baseListener, nil
}

func (b *BaseListener) Accept() (net.Conn, error) {
	return b.listener.Accept()
}

// Address implements constant.InboundListener
func (b *BaseListener) Address() string {
	return b.addr
}

// Close implements constant.InboundListener
func (*BaseListener) Close() error {
	return nil
}

func (b *BaseListener) Closed() bool {
	return b.closed
}
