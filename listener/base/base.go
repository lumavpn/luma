package base

import (
	"net"
	"sync"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy/inbound"
)

type BaseListener struct {
	listener   net.Listener
	addr       string
	closed     bool
	listenAddr string
	mu         sync.Mutex
	tunnel     adapter.TransportHandler
}

type BaseOptions struct {
	Addr     string
	Listener net.Listener
	Tunnel   adapter.TransportHandler
}

func New(opts BaseOptions) (*BaseListener, error) {
	if opts.Addr == "" {
		opts.Addr = ":0"
	}

	baseListener := &BaseListener{
		addr:   opts.Addr,
		tunnel: opts.Tunnel,
	}
	return baseListener, nil
}

func (b *BaseListener) Accept() (net.Conn, error) {
	return b.listener.Accept()
}

func (b *BaseListener) ListenTCP() error {
	l, err := inbound.Listen("tcp", b.addr)
	if err != nil {
		return err
	}
	log.Debugf("Inbound listen %s", l.Addr().String())
	b.mu.Lock()
	b.listener = l
	b.listenAddr = l.Addr().String()
	b.mu.Unlock()
	return nil
}

// Address implements constant.InboundListener
func (b *BaseListener) Address() string {
	if b.listenAddr != "" {
		return b.listenAddr
	}
	return b.addr
}

// Close implements constant.InboundListener
func (*BaseListener) Close() error {
	return nil
}

func (b *BaseListener) Closed() bool {
	return b.closed
}
