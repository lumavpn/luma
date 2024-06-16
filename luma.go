package luma

import (
	"context"
	"sync"

	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/ipfilter"
	"github.com/lumavpn/luma/listener/inbound"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/tunnel"
)

type Luma struct {
	config    *config.Config
	listeners map[string]inbound.InboundListener
	proxies   map[string]proxy.Proxy
	mu        sync.Mutex
	tunnel    tunnel.Tunnel
}

// New creates a new instance of Luma
func New(cfg *config.Config) *Luma {
	return &Luma{
		config: cfg,
		tunnel: tunnel.New(),
	}
}

// applyConfig applies the given Config to the instance of Luma to complete setup
func applyConfig(cfg *config.Config) error {
	ipfilter.SetAllowedIPs(cfg.LanAllowedIPs)
	ipfilter.SetDisAllowedIPs(cfg.LanDisAllowedIPs)
	return nil
}

// Start starts the default engine running Luma. If there is any issue with the setup process, an error is returned
func (lu *Luma) Start(ctx context.Context) error {
	log.Debug("Starting new instance")
	cfg := lu.config
	proxies, err := parseProxies(cfg)
	if err != nil {
		return err
	}
	listeners, err := parseListeners(cfg)
	if err != nil {
		return err
	}
	lu.mu.Lock()
	lu.listeners = listeners
	lu.proxies = proxies
	lu.mu.Unlock()

	return applyConfig(cfg)
}

// Stop stops running the Luma engine
func (lu *Luma) Stop() {

}
