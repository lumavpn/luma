package luma

import (
	"context"
	"sync"

	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/ipfilter"
	"github.com/lumavpn/luma/listener/inbound"
	"github.com/lumavpn/luma/listener/socks"
	"github.com/lumavpn/luma/listener/tun"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxydialer"
	"github.com/lumavpn/luma/tunnel"
)

type Luma struct {
	// config is the configuration this instance of Luma is using
	config *config.Config

	// proxies is a map of proxies that Luma is configured to proxy traffic through
	proxies map[string]proxy.Proxy

	// listeners are inbound listeners that this instance of Luma is configured with
	listeners map[string]inbound.InboundListener

	socksListener    *socks.Listener
	socksUDPListener *socks.UDPListener
	tunListener      *tun.Listener

	mu     sync.Mutex
	tunnel tunnel.Tunnel
}

// New creates a new instance of Luma
func New(cfg *config.Config) *Luma {
	proxyDialer := proxydialer.New()
	return &Luma{
		config: cfg,
		tunnel: tunnel.New(proxyDialer),
	}
}

// applyConfig applies the given Config to the instance of Luma to complete setup
func (lu *Luma) applyConfig(cfg *config.Config) error {
	ipfilter.SetAllowedIPs(cfg.LanAllowedIPs)
	ipfilter.SetDisAllowedIPs(cfg.LanDisAllowedIPs)
	if err := lu.setupLocalSocks(cfg); err != nil {
		return err
	}
	if err := lu.setupTun(cfg); err != nil {
		return err
	}
	return nil
}

// Start starts the default engine running Luma. If there is any issue with the setup process, an error is returned
func (lu *Luma) Start(ctx context.Context) error {
	log.Debug("Starting new instance")
	cfg := lu.config
	if err := lu.parseConfig(cfg); err != nil {
		return err
	}
	return lu.applyConfig(cfg)
}

// Stop stops running the Luma engine
func (lu *Luma) Stop() {
	if lu.socksListener != nil {
		lu.socksListener.Close()
	}
	if lu.socksUDPListener != nil {
		lu.socksUDPListener.Close()
	}
	if lu.tunListener != nil {
		lu.tunListener.Close()
	}
}
