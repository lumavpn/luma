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
	"github.com/lumavpn/luma/rule"
	"github.com/lumavpn/luma/tunnel"
)

type Luma struct {
	// config is the configuration this instance of Luma is using
	config *config.Config

	// proxies is a map of proxies that Luma is configured to proxy traffic through
	proxies map[string]proxy.Proxy

	proxyDialer proxydialer.ProxyDialer

	rules []rule.Rule

	// listeners are inbound listeners that this instance of Luma is configured with
	listeners map[string]inbound.InboundListener

	socksListener    *socks.Listener
	socksUDPListener *socks.UDPListener
	tunListener      *tun.Listener

	mu     sync.Mutex
	tunnel tunnel.Tunnel
}

// New creates a new instance of Luma
func New(cfg *config.Config) (*Luma, error) {
	proxies, err := parseProxies(cfg)
	if err != nil {
		return nil, err
	}
	listeners, err := parseListeners(cfg)
	if err != nil {
		return nil, err
	}

	rules, err := parseRules(cfg, proxies)
	if err != nil {
		return nil, err
	}

	log.Debugf("Have %d rules", len(rules))
	proxyDialer := proxydialer.New(proxies, rules)
	return &Luma{
		config:      cfg,
		listeners:   listeners,
		rules:       rules,
		proxies:     proxies,
		proxyDialer: proxyDialer,
		tunnel:      tunnel.New(proxyDialer),
	}, nil
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
	return lu.applyConfig(lu.config)
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
