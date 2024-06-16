package luma

import (
	"context"
	"sync"

	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
)

type Luma struct {
	config  *config.Config
	proxies map[string]proxy.Proxy
	mu      sync.Mutex
}

// New creates a new instance of Luma
func New(cfg *config.Config) *Luma {
	return &Luma{
		config: cfg,
	}
}

// Start starts the default engine running Luma. If there is any issue with the setup process, an error is returned
func (lu *Luma) Start(ctx context.Context) error {
	log.Debug("Starting new instance")
	proxies, err := parseProxies(lu.config)
	if err != nil {
		return err
	}
	lu.mu.Lock()
	lu.proxies = proxies
	lu.mu.Unlock()
	return nil
}

// Stop stops running the Luma engine
func (lu *Luma) Stop() {

}
