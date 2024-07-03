package luma

import (
	"context"
	"runtime"
	"sync"

	"github.com/lumavpn/luma/component/ebpf"
	"github.com/lumavpn/luma/component/resolver"
	"github.com/lumavpn/luma/component/trie"
	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/dns"
	"github.com/lumavpn/luma/listener/inner"
	"github.com/lumavpn/luma/listener/tun"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/provider"
	"github.com/lumavpn/luma/proxydialer"
	"github.com/lumavpn/luma/rule"
	"github.com/lumavpn/luma/stack"
	"github.com/lumavpn/luma/tunnel"
	SNI "github.com/lumavpn/luma/tunnel/sniffer"
)

type Luma struct {
	// config is the configuration this instance of Luma is using
	config *config.Config
	// proxies is a map of proxies that Luma is configured to proxy traffic through
	proxies     map[string]proxy.Proxy
	providers   map[string]provider.ProxyProvider
	proxyDialer proxydialer.ProxyDialer
	// Tunnel
	stack       stack.Stack
	tunListener *tun.Listener
	tunnel      tunnel.Tunnel

	autoRedirProgram *ebpf.TcEBpfProgram
	tcProgram        *ebpf.TcEBpfProgram
	lastTunConfig    config.Tun

	// Rules
	rules         []rule.Rule
	ruleProviders map[string]provider.RuleProvider
	subRules      map[string][]rule.Rule

	// DNS
	hosts *trie.DomainTrie[resolver.HostValue]

	mu      sync.Mutex
	tcMux   sync.Mutex
	tunMu   sync.Mutex
	started bool
}

// New creates a new instance of Luma
func New(cfg *config.Config) (*Luma, error) {
	proxyDialer := proxydialer.New()
	return &Luma{
		config:      cfg,
		proxyDialer: proxyDialer,
		proxies:     make(map[string]proxy.Proxy),
		tunnel:      tunnel.New(proxyDialer),
	}, nil
}

// Start starts the default engine running Luma. If there is any issue with the setup process, an error is returned
func (lu *Luma) Start(ctx context.Context) error {
	log.Debug("Starting new instance")
	cfg := lu.config
	if err := lu.parseConfig(cfg); err != nil {
		return err
	}

	if err := lu.applyConfig(ctx, cfg); err != nil {
		return err
	}
	log.Debug("Luma successfully started")
	lu.SetStarted(true)
	return nil
}

func (lu *Luma) updateGeneral(ctx context.Context, cfg *config.Config) {
	lu.tunnel.SetMode(cfg.Mode)
}

func (lu *Luma) SetStarted(started bool) {
	lu.mu.Lock()
	defer lu.mu.Unlock()
	lu.started = started
}

// Stop stops running the Luma engine
func (lu *Luma) Stop() error {
	if lu.tunListener != nil {
		log.Debug("Closing tun device")
		lu.tunListener.Close()
		lu.tunListener = nil
	}

	//tproxy.CleanupTProxyIPTables()
	//resolver.StoreFakePoolState()
	return nil
}

func (lu *Luma) updateTun(ctx context.Context, cfg *config.Tun) error {
	log.Debug("Recreating tunnel")
	if err := lu.startTunListener(ctx, cfg); err != nil {
		return err
	}
	return nil
}

// applyConfig applies the given Config to the instance of Luma to complete setup
func (lu *Luma) applyConfig(ctx context.Context, cfg *config.Config) error {
	lu.updateSniffer(cfg.Sniffer)
	updateHosts(lu.hosts)

	lu.updateGeneral(ctx, cfg)
	lu.updateDNS(cfg.DNS, lu.ruleProviders)
	/*if err := lu.updateListeners(ctx, cfg, lu.listeners, true); err != nil {
		return err
	}*/
	if err := lu.updateTun(ctx, cfg.Tun); err != nil {
		return err
	}
	lu.tunnel.OnInnerLoading()
	initInnerTcp(lu.tunnel)
	provider.LoadProxyProvider(lu.providers)
	//updateProfile(cfg, lu.proxies)
	provider.LoadRuleProvider(lu.ruleProviders)
	runtime.GC()
	lu.tunnel.OnRunning()
	provider.HCCompatibleProvider(lu.providers)
	log.Debug("Finished applying config")
	return nil
}

func updateHosts(hosts *trie.DomainTrie[resolver.HostValue]) {
	resolver.DefaultHosts = resolver.NewHosts(hosts)
}

func initInnerTcp(tunnel tunnel.Tunnel) {
	inner.New(tunnel)
}

func (lu *Luma) updateDNS(c *config.DNS, ruleProvider map[string]provider.RuleProvider) {
	if !c.Enable {
		resolver.DefaultResolver = nil
		resolver.DefaultHostMapper = nil
		resolver.DefaultLocalServer = nil
		dns.ReCreateServer("", nil, nil)
		return
	}
	log.Debug("Updating dns")
	cfg := dns.Config{
		Main:         c.NameServer,
		Fallback:     c.Fallback,
		IPv6:         c.IPv6,
		IPv6Timeout:  c.IPv6Timeout,
		EnhancedMode: c.EnhancedMode,
		Pool:         c.FakeIPRange,
		Hosts:        c.Hosts,
		FallbackFilter: dns.FallbackFilter{
			GeoIP:     c.FallbackFilter.GeoIP,
			GeoIPCode: c.FallbackFilter.GeoIPCode,
			IPCIDR:    c.FallbackFilter.IPCIDR,
			Domain:    c.FallbackFilter.Domain,
			GeoSite:   c.FallbackFilter.GeoSite,
		},
		Default:        c.DefaultNameserver,
		Policy:         c.NameServerPolicy,
		ProxyServer:    c.ProxyServerNameserver,
		RuleProviders:  ruleProvider,
		CacheAlgorithm: c.CacheAlgorithm,
	}

	r := dns.NewResolver(cfg, lu.proxyDialer)
	pr := dns.NewProxyServerHostResolver(r)
	m := dns.NewEnhancer(cfg)

	// reuse cache of old host mapper
	if old := resolver.DefaultHostMapper; old != nil {
		m.PatchFrom(old.(*dns.ResolverEnhancer))
	}

	resolver.DefaultResolver = r
	resolver.DefaultHostMapper = m
	resolver.DefaultLocalServer = dns.NewLocalServer(r, m)
	resolver.UseSystemHosts = c.UseSystemHosts

	if pr.Invalid() {
		resolver.ProxyServerHostResolver = pr
	}

	dns.ReCreateServer(c.Listen, r, m)
}

func (lu *Luma) updateSniffer(sniffer *config.Sniffer) {
	if sniffer == nil {
		return
	}
	if sniffer.Enable {
		dispatcher, err := SNI.NewSnifferDispatcher(
			sniffer.Sniffers, sniffer.ForceDomain, sniffer.SkipDomain,
			sniffer.ForceDnsMapping, sniffer.ParsePureIp,
		)
		if err != nil {
			log.Warnf("initial sniffer failed, err:%v", err)
		}

		lu.tunnel.UpdateSniffer(dispatcher)
		log.Info("Sniffer is loaded and working")
	} else {
		dispatcher, err := SNI.NewCloseSnifferDispatcher()
		if err != nil {
			log.Warnf("initial sniffer failed, err:%v", err)
		}

		lu.tunnel.UpdateSniffer(dispatcher)
		log.Info("Sniffer is closed")
	}
}
