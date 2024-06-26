package luma

import (
	"context"
	"net"
	"sync"

	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/stack"
	"github.com/lumavpn/luma/tunnel"
)

type Luma struct {
	// config is the configuration this instance of Luma is using
	config *config.Config
	// proxies is a map of proxies that Luma is configured to proxy traffic through
	proxies map[string]proxy.Proxy

	stack stack.Stack
	// Tunnel
	tunnel tunnel.Tunnel

	mu sync.Mutex
}

// New creates a new instance of Luma
func New(cfg *config.Config) (*Luma, error) {
	return &Luma{
		config: cfg,
		tunnel: tunnel.New(),
	}, nil
}

// Start starts the default engine running Luma. If there is any issue with the setup process, an error is returned
func (lu *Luma) Start(ctx context.Context) error {
	log.Debug("Starting new instance")
	stack, err := stack.New(&stack.Options{
		Handler: lu,
	})
	if err != nil {
		return err
	}
	lu.SetStack(stack)
	return lu.applyConfig(lu.config)
}

func (lu *Luma) SetStack(s stack.Stack) {
	lu.mu.Lock()
	lu.stack = s
	lu.mu.Unlock()
}

func (lu *Luma) NewConnection(ctx context.Context, conn net.Conn, m *metadata.Metadata) error {
	return nil
}

func (lu *Luma) NewPacketConnection(ctx context.Context, conn net.PacketConn, m *metadata.Metadata) error {
	return nil
}

// Stop stops running the Luma engine
func (lu *Luma) Stop() {

}

// applyConfig applies the given Config to the instance of Luma to complete setup
func (lu *Luma) applyConfig(cfg *config.Config) error {
	return nil
}
