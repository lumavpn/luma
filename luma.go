package luma

import (
	"context"
	"net"
	"net/netip"
	"sync"

	"github.com/lumavpn/luma/component/trie"
	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/dns/resolver"
	"github.com/lumavpn/luma/listener/tun"
	"github.com/lumavpn/luma/local"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxydialer"
	"github.com/lumavpn/luma/stack"
	"github.com/lumavpn/luma/tunnel"
	"github.com/lumavpn/luma/util"
)

type Luma struct {
	// config is the configuration this instance of Luma is using
	config *config.Config

	localServers map[string]local.LocalServer
	proxyDialer  proxydialer.ProxyDialer
	// proxies is a map of proxies that Luma is configured to proxy traffic through
	proxies map[string]proxy.Proxy

	stack stack.Stack
	// Tunnel
	dnsAdds       []netip.AddrPort
	hosts         *trie.DomainTrie[resolver.HostValue]
	tunName       string
	tunListener   *tun.Listener
	lastTunConfig config.Tun
	tunMu         sync.Mutex
	tunnel        tunnel.Tunnel

	mu sync.Mutex

	socksServer *local.SocksServer
}

// New creates a new instance of Luma
func New(cfg *config.Config) *Luma {
	proxyDialer := proxydialer.New()
	return &Luma{
		config:       cfg,
		proxyDialer:  proxyDialer,
		localServers: make(map[string]local.LocalServer),
		proxies:      make(map[string]proxy.Proxy),
		tunnel:       tunnel.New(proxyDialer),
	}
}

// Start starts the default engine running Luma. If there is any issue with the setup process, an error is returned
func (lu *Luma) Start(ctx context.Context) error {
	cfg := lu.config
	proxies, localServers, err := lu.parseConfig(cfg)
	if err != nil {
		return err
	}
	lu.mu.Lock()
	lu.proxies = proxies
	lu.localServers = localServers
	lu.mu.Unlock()
	lu.proxyDialer.SetProxies(proxies)

	lu.tunnel.SetMode(cfg.Mode)

	go lu.startLocal(localServers, true)
	if err := lu.localSocksServer(cfg); err != nil {
		return err
	}

	if err := lu.updateDNS(cfg.DNS); err != nil {
		return err
	}

	return lu.startEngine(ctx)
}

func (lu *Luma) startEngine(ctx context.Context) error {
	log.Debug("Starting new instance")

	cfg := lu.config
	tunMTU := cfg.MTU
	if tunMTU == 0 {
		tunMTU = 9000
	}

	if cfg.Interface != "" {
		log.Debugf("Setting default interface to %s", cfg.Interface)
		iface, err := net.InterfaceByName(cfg.Interface)
		if err != nil {
			return err
		}
		dialer.DefaultInterface.Store(iface.Name)
		dialer.DefaultInterfaceIndex.Store(int32(iface.Index))
		log.Infof("bind to interface: %s", cfg.Interface)
	}

	tunName := cfg.Device
	if tunName == "" || !checkTunName(tunName) {
		tunName = util.CalculateInterfaceName("Luma")
		log.Debugf("Setting tun device name to %s", tunName)
		cfg.Device = tunName
	}

	if err := lu.startTunListener(ctx, cfg.Tun); err != nil {
		return err
	}

	log.Debug("Luma successfully started")
	return nil
}

func (lu *Luma) SetDnsAdds(dnsAdds []netip.AddrPort) {
	lu.mu.Lock()
	lu.dnsAdds = dnsAdds
	lu.mu.Unlock()
}

func (lu *Luma) SetStack(s stack.Stack) {
	lu.mu.Lock()
	lu.stack = s
	lu.mu.Unlock()
}

// Stop stops running the Luma engine
func (lu *Luma) Stop() {
	log.Debug("Stopping luma..")
	if lu.stack != nil {
		lu.stack.Close()
	}
	if lu.socksServer != nil {
		lu.socksServer.Stop()
	}
}

func ShouldIgnorePacketError(err error) bool {
	return false
}
