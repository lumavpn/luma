package luma

import (
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"runtime"
	"sync"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/component/ca"
	"github.com/lumavpn/luma/component/ebpf"
	"github.com/lumavpn/luma/component/iface"
	"github.com/lumavpn/luma/component/profile"
	"github.com/lumavpn/luma/component/profile/cachefile"
	"github.com/lumavpn/luma/component/resolver"
	"github.com/lumavpn/luma/component/trie"
	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/dns"
	"github.com/lumavpn/luma/features"
	"github.com/lumavpn/luma/listener"
	"github.com/lumavpn/luma/listener/autoredir"
	"github.com/lumavpn/luma/listener/http"
	IN "github.com/lumavpn/luma/listener/inbound"
	"github.com/lumavpn/luma/listener/inner"
	"github.com/lumavpn/luma/listener/mixed"
	"github.com/lumavpn/luma/listener/redir"
	"github.com/lumavpn/luma/listener/socks"
	"github.com/lumavpn/luma/listener/tproxy"
	"github.com/lumavpn/luma/listener/tun"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
	PA "github.com/lumavpn/luma/proxy/adapter"
	"github.com/lumavpn/luma/proxy/inbound"
	"github.com/lumavpn/luma/proxy/outboundgroup"
	"github.com/lumavpn/luma/proxy/provider"
	"github.com/lumavpn/luma/proxydialer"
	"github.com/lumavpn/luma/rule"
	"github.com/lumavpn/luma/stack"
	"github.com/lumavpn/luma/sysproxy"
	"github.com/lumavpn/luma/tunnel"
	SNI "github.com/lumavpn/luma/tunnel/sniffer"
)

var (
	mu          sync.Mutex
	defaultLuma *Luma
)

type Luma struct {
	config  *config.Config
	started bool

	// dns
	hosts *trie.DomainTrie[resolver.HostValue]

	// proxies
	groupsList    *list.List
	proxiesList   *list.List
	proxies       map[string]proxy.Proxy
	providers     map[string]provider.ProxyProvider
	rules         []rule.Rule
	ruleProviders map[string]provider.RuleProvider
	subRules      map[string][]rule.Rule

	// listeners
	listeners           map[string]IN.InboundListener
	httpListener        *http.Listener
	mixedListener       *mixed.Listener
	mixedUDPLister      *socks.UDPListener
	redirListener       *redir.Listener
	redirUDPListener    *tproxy.UDPListener
	shadowSocksListener listener.MultiAddrListener
	socksListener       *socks.Listener
	socksUDPListener    *socks.UDPListener
	tproxyListener      *tproxy.Listener
	tproxyUDPListener   *tproxy.UDPListener

	proxy       proxy.Proxy
	proxyDialer proxydialer.ProxyDialer

	// tunnel
	inboundListeners map[string]IN.InboundListener
	lastTunConfig    config.Tun
	stack            stack.Stack
	tunListener      *tun.Listener
	tunnel           tunnel.Tunnel

	autoRedirListener *autoredir.Listener
	autoRedirProgram  *ebpf.TcEBpfProgram
	tcProgram         *ebpf.TcEBpfProgram

	autoRedirMu sync.Mutex
	httpMu      sync.Mutex
	inboundMux  sync.Mutex
	mixedMu     sync.Mutex
	mu          sync.Mutex
	redirMu     sync.Mutex
	socksMu     sync.Mutex
	ssMu        sync.Mutex
	tcMux       sync.Mutex
	tproxyMu    sync.Mutex
	tunMu       sync.Mutex
}

type Options struct {
	Config      *config.Config
	ProxyDialer proxydialer.ProxyDialer
}

func New(cfg *config.Config) *Luma {
	proxyDialer := proxydialer.New()
	tunnel := tunnel.New(proxyDialer)
	return &Luma{
		config:           cfg,
		inboundListeners: make(map[string]IN.InboundListener),
		proxyDialer:      proxyDialer,
		proxies:          make(map[string]proxy.Proxy),
		providers:        make(map[string]provider.ProxyProvider),
		tunnel:           tunnel,
	}
}

// Start starts the default engine running Luma
func (lu *Luma) Start(ctx context.Context) error {
	lu.mu.Lock()
	started := lu.started
	lu.mu.Unlock()
	if started {
		return errors.New("Luma already started")
	}
	if err := lu.ApplyConfig(ctx, lu.config); err != nil {
		log.Errorf("Error applying config: %v", err)
		return err
	}
	log.Debug("Luma successfully started")
	lu.SetStarted(true)
	return nil
}

func (lu *Luma) Config() *config.Config {
	return lu.config
}

func (lu *Luma) Dialer() proxydialer.ProxyDialer {
	return lu.proxyDialer
}

func (lu *Luma) SetStarted(started bool) {
	lu.mu.Lock()
	defer lu.mu.Unlock()
	lu.started = started
}

func (lu *Luma) parseConfig(cfg *config.Config) error {
	proxies, providers, err := parseProxies(cfg, lu.proxyDialer)
	if err != nil {
		return err
	}
	lu.SetProxies(proxies)
	lu.SetProxyProviders(providers)

	log.Debugf("Have %d proxies", len(proxies))

	listeners, err := parseListeners(cfg)
	if err != nil {
		return err
	}
	lu.SetListeners(listeners)
	log.Debugf("Have %d listeners", len(listeners))

	ruleProviders, err := parseRuleProviders(cfg.RuleProviders)
	if err != nil {
		return err
	}
	lu.SetRuleProviders(ruleProviders)
	subRules, err := parseSubRules(cfg.SubRules, proxies)
	if err != nil {
		return err
	}

	rules, err := parseRules(cfg.Rules, proxies, subRules, "rules")
	if err != nil {
		return err
	}

	lu.SetRules(rules)
	lu.SetSubRules(subRules)

	log.Debugf("Have %d rules", len(rules))
	hosts, err := parseHosts(cfg.Hosts)
	if err != nil {
		return err
	}
	lu.SetHosts(hosts)

	cfg.DNS, err = parseDNS(cfg, lu.tunnel, hosts, rules, ruleProviders)
	if err != nil {
		return err
	}

	cfg.Tun, err = parseTun(cfg, lu.tunnel)
	if err != nil {
		log.Fatalf("unable to parse tun config: %v", err)
	}

	cfg.Tun.RedirectToTun = cfg.EBpf.RedirectToTun

	lu.updateProxies(proxies, providers)
	lu.updateRules(rules, subRules, ruleProviders)

	return nil
}

func (lu *Luma) ApplyConfig(ctx context.Context, cfg *config.Config) error {
	log.Debug("Calling apply config")

	b, _ := json.Marshal(cfg)
	log.Debugf("Config is %s", string(b))

	tunnel := lu.tunnel

	tunnel.OnSuspend()

	ca.ResetCertificate()

	if err := lu.parseConfig(cfg); err != nil {
		return err
	}

	lu.updateSniffer(cfg.Sniffer)
	updateHosts(lu.hosts)

	lu.updateGeneral(ctx, cfg)
	lu.updateDNS(cfg.DNS, lu.ruleProviders)
	if err := lu.updateListeners(ctx, cfg, lu.listeners, true); err != nil {
		return err
	}
	if err := lu.updateTun(ctx, cfg.Tun); err != nil {
		return err
	}
	tunnel.OnInnerLoading()
	initInnerTcp(lu.tunnel)
	provider.LoadProxyProvider(lu.providers)
	updateProfile(cfg, lu.proxies)
	provider.LoadRuleProvider(lu.ruleProviders)
	runtime.GC()
	tunnel.OnRunning()
	provider.HCCompatibleProvider(lu.providers)
	log.Debug("Finished applying config")
	return nil
}

func (lu *Luma) SetHosts(tree *trie.DomainTrie[resolver.HostValue]) {
	lu.mu.Lock()
	defer lu.mu.Unlock()
	lu.hosts = tree
}

func updateHosts(hosts *trie.DomainTrie[resolver.HostValue]) {
	resolver.DefaultHosts = resolver.NewHosts(hosts)
}

func (lu *Luma) updateTun(ctx context.Context, cfg *config.Tun) error {
	log.Debug("Recreating tunnel")
	if err := lu.startTunListener(ctx, cfg); err != nil {
		return err
	}
	if err := lu.recreateRedirToTun(cfg.RedirectToTun, cfg.Device); err != nil {
		log.Error(err)
	}
	return nil
}

func initInnerTcp(tunnel tunnel.Tunnel) {
	inner.New(tunnel)
}

func (lu *Luma) patchInboundListeners(newListenerMap map[string]IN.InboundListener, tunnel adapter.TransportHandler, dropOld bool) {
	lu.inboundMux.Lock()
	defer lu.inboundMux.Unlock()

	for name, newListener := range newListenerMap {
		if oldListener, ok := lu.inboundListeners[name]; ok {
			if !oldListener.Config().Equal(newListener.Config()) {
				_ = oldListener.Close()
			} else {
				continue
			}
		}
		if err := newListener.Listen(tunnel); err != nil {
			log.Errorf("Listener %s listen err: %s", name, err.Error())
			continue
		}
		lu.inboundListeners[name] = newListener
	}

	if dropOld {
		for name, oldListener := range lu.inboundListeners {
			if _, ok := newListenerMap[name]; !ok {
				_ = oldListener.Close()
				delete(lu.inboundListeners, name)
			}
		}
	}
}

func (lu *Luma) updateGeneral(ctx context.Context, cfg *config.Config) {
	level, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Error(err)
	} else {
		log.SetLevel(level)
	}
	lu.tunnel.SetMode(cfg.Mode)
	lu.proxyDialer.SetMode(cfg.Mode)

	if cfg.TCPConcurrent {
		dialer.SetTcpConcurrent(cfg.TCPConcurrent)
		log.Info("Use tcp concurrent")
	}

	inbound.SetTfo(cfg.InboundTfo)
	inbound.SetMPTCP(cfg.InboundMPTCP)

	resolver.DisableIPv6 = !cfg.IPv6
	log.Debugf("Setting default interface to %s", cfg.Interface)
	dialer.DefaultInterface.Store(cfg.Interface)
	dialer.DefaultRoutingMark.Store(int32(cfg.Mark))
	if cfg.Mark > 0 {
		log.Infof("Use routing mark: %#x", cfg.Mark)
	}
	iface.FlushCache()
}

func (lu *Luma) updateListeners(ctx context.Context, cfg *config.Config, listeners map[string]IN.InboundListener, force bool) error {
	log.Debug("Updating listeners")

	tunnel := lu.tunnel

	lu.patchInboundListeners(listeners, tunnel, true)
	if !force {
		return nil
	}

	inbound.SetSkipAuthPrefixes(cfg.SkipAuthPrefixes)
	inbound.SetAllowedIPs(cfg.LanAllowedIPs)
	inbound.SetDisAllowedIPs(cfg.LanDisAllowedIPs)
	if err := lu.recreateHTTP(ctx, cfg, tunnel); err != nil {
		return err
	}
	if err := lu.recreateSocks(ctx, cfg, tunnel); err != nil {
		return err
	}
	if err := lu.recreateRedir(ctx, cfg, tunnel); err != nil {
		return err
	}
	if !features.CMFA {
		if err := lu.recreateAutoRedir(cfg.EBpf.AutoRedir, tunnel); err != nil {
			log.Error(err)
		}
	}
	if err := lu.recreateTProxy(cfg, tunnel); err != nil {
		return err
	}
	if err := lu.recreateMixed(cfg, tunnel); err != nil {
		return err
	}

	if cfg.MixedPort > 0 && cfg.EnableSystemProxy {
		return sysproxy.EnableAll(cfg.MixedPort)
	}

	return nil
}

func (lu *Luma) updateProxies(proxies map[string]proxy.Proxy, providers map[string]provider.ProxyProvider) {
	log.Debugf("Updating proxies, have %d proxies", len(proxies))
	lu.proxyDialer.UpdateProxies(proxies, providers)
}

func (lu *Luma) updateRules(rules []rule.Rule, subRules map[string][]rule.Rule, ruleProviders map[string]provider.RuleProvider) {
	lu.proxyDialer.UpdateRules(rules, subRules, ruleProviders)
}

func updateProfile(cfg *config.Config, proxies map[string]proxy.Proxy) {
	profileCfg := cfg.Profile

	profile.StoreSelected.Store(profileCfg.StoreSelected)
	if profileCfg.StoreSelected {
		patchSelectGroup(proxies)
	}
}

func patchSelectGroup(proxies map[string]proxy.Proxy) {
	mapping := cachefile.Cache().SelectedMap()
	if mapping == nil {
		return
	}

	for name, proxy := range proxies {
		outbound, ok := proxy.(*PA.Proxy)
		if !ok {
			continue
		}

		selector, ok := outbound.ProxyAdapter.(outboundgroup.SelectAble)
		if !ok {
			continue
		}

		selected, exist := mapping[name]
		if !exist {
			continue
		}

		selector.ForceSet(selected)
	}
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

func (lu *Luma) Stop() error {
	log.Debug("Stopping luma..")

	cfg := lu.config

	defer lu.SetStarted(false)

	if lu.autoRedirListener != nil {
		lu.autoRedirListener.Close()
		lu.autoRedirListener = nil
	}

	if lu.httpListener != nil {
		lu.httpListener.Close()
		lu.httpListener = nil
	}

	if lu.mixedListener != nil && cfg.EnableSystemProxy {
		log.Debug("Disabling system proxy")
		sysproxy.Disable()
	}

	if lu.mixedListener != nil {
		lu.mixedListener.Close()
		lu.mixedListener = nil
	}

	if lu.redirListener != nil {
		lu.redirListener.Close()
		lu.redirListener = nil
	}

	if lu.shadowSocksListener != nil {
		lu.shadowSocksListener.Close()
		lu.shadowSocksListener = nil
	}

	if lu.socksListener != nil {
		lu.socksListener.Close()
		lu.socksListener = nil
	}

	if lu.socksUDPListener != nil {
		lu.socksUDPListener.Close()
		lu.socksUDPListener = nil
	}

	if lu.tunListener != nil {
		log.Debug("Closing tun device")
		lu.tunListener.Close()
		lu.tunListener = nil
	}

	tproxy.CleanupTProxyIPTables()
	resolver.StoreFakePoolState()

	return nil
}

func (lu *Luma) SetSocksListener(server *socks.Listener) {
	lu.socksMu.Lock()
	defer lu.socksMu.Unlock()
	lu.socksListener = server
}

func (lu *Luma) SetSocksUDPListener(server *socks.UDPListener) {
	lu.socksMu.Lock()
	defer lu.socksMu.Unlock()
	lu.socksUDPListener = server
}
