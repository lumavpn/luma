package luma

import (
	"context"
	"sync"

	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/tunnel"
)

type Luma struct {
	// config is the configuration this instance of Luma is using
	config *config.Config
	// proxies is a map of proxies that Luma is configured to proxy traffic through
	proxies map[string]proxy.Proxy

	// Tunnel
	tunnel.Tunnel

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
	return lu.applyConfig(lu.config)
}

// Stop stops running the Luma engine
func (lu *Luma) Stop() {

}

// applyConfig applies the given Config to the instance of Luma to complete setup
func (lu *Luma) applyConfig(cfg *config.Config) error {
	return nil
}
