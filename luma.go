package luma

import (
	"context"
	"fmt"
	"sync"

	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/ipfilter"
	"github.com/lumavpn/luma/listener/inbound"
	"github.com/lumavpn/luma/listener/socks"
	"github.com/lumavpn/luma/listener/tun"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/tunnel"
)

type Luma struct {
	config *config.Config

	proxies map[string]proxy.Proxy

	listeners map[string]inbound.InboundListener

	socksListener    *socks.Listener
	socksUDPListener *socks.UDPListener
	tunListener      *tun.Listener

	mu     sync.Mutex
	tunnel tunnel.Tunnel
}

// New creates a new instance of Luma
func New(cfg *config.Config) *Luma {
	return &Luma{
		config: cfg,
		tunnel: tunnel.New(),
	}
}

func (lu *Luma) setupLocalSocks(cfg *config.Config) error {
	addr := fmt.Sprintf("127.0.0.1:%d", cfg.SocksPort)
	tcpListener, err := socks.New(addr, lu.tunnel)
	if err != nil {
		return err
	}

	udpListener, err := socks.NewUDP(addr, lu.tunnel)
	if err != nil {
		tcpListener.Close()
		return err
	}

	lu.mu.Lock()
	lu.socksListener = tcpListener
	lu.socksUDPListener = udpListener
	lu.mu.Unlock()

	log.Debugf("SOCKS proxy listening at: %s", tcpListener.Address())
	return nil
}

func (lu *Luma) setupTun(cfg *config.Tun) error {
	listener, err := tun.New(cfg, lu.tunnel)
	if err != nil {
		return err
	}
	lu.mu.Lock()
	lu.tunListener = listener
	lu.mu.Unlock()
	return nil
}

// applyConfig applies the given Config to the instance of Luma to complete setup
func (lu *Luma) applyConfig(cfg *config.Config) error {
	ipfilter.SetAllowedIPs(cfg.LanAllowedIPs)
	ipfilter.SetDisAllowedIPs(cfg.LanDisAllowedIPs)
	if err := lu.setupLocalSocks(cfg); err != nil {
		return err
	}
	tunConfig, err := parseTun(cfg, lu.tunnel)
	if err != nil {
		return err
	}
	if err := lu.setupTun(tunConfig); err != nil {
		return err
	}
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
}
