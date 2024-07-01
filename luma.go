package luma

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"

	"github.com/lumavpn/luma/component/iface"
	"github.com/lumavpn/luma/component/trie"
	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/dns/resolver"
	"github.com/lumavpn/luma/listener/mux"
	"github.com/lumavpn/luma/local"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/proxydialer"
	"github.com/lumavpn/luma/stack"
	"github.com/lumavpn/luma/stack/tun"
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
	device  tun.Tun
	dnsAdds []netip.AddrPort
	hosts   *trie.DomainTrie[resolver.HostValue]
	tunName string
	tunnel  tunnel.Tunnel

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
	lu.flushDefaultInterface()

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

	var dnsAdds []netip.AddrPort

	for _, d := range cfg.Tun.DNSHijack {
		if _, after, ok := strings.Cut(d, "://"); ok {
			d = after
		}
		d = strings.Replace(d, "any", "0.0.0.0", 1)
		addrPort, err := netip.ParseAddrPort(d)
		if err != nil {
			return fmt.Errorf("parse dns-hijack url error: %w", err)
		}

		dnsAdds = append(dnsAdds, addrPort)
	}
	lu.SetDnsAdds(dnsAdds)

	h, err := mux.NewListenerHandler(mux.ListenerConfig{
		Tunnel: lu.tunnel,
		Proto:  proto.Proto_TUN,
		//Additions: additions,
	})
	if err != nil {
		return err
	}

	tunAddressPrefix := netip.MustParsePrefix("198.18.0.1/16")
	tunAddressPrefix = netip.PrefixFrom(tunAddressPrefix.Addr(), 30)

	tunOptions := tun.Options{
		AutoRoute:                cfg.Tun.AutoRoute,
		Name:                     cfg.Device,
		MTU:                      tunMTU,
		WireGuard:                true,
		Inet4RouteAddress:        cfg.Tun.Inet4RouteAddress,
		Inet6RouteAddress:        cfg.Tun.Inet6RouteAddress,
		Inet4Address:             cfg.Tun.Inet4Address,
		Inet6Address:             cfg.Tun.Inet6Address,
		Inet4RouteExcludeAddress: cfg.Tun.Inet4RouteExcludeAddress,
		Inet6RouteExcludeAddress: cfg.Tun.Inet6RouteExcludeAddress,
	}
	if len(cfg.Tun.Inet4Address) == 0 {
		tunOptions.Inet4Address = []netip.Prefix{tunAddressPrefix}
	}
	if !cfg.IPv6 || !verifyIP6() {
		tunOptions.Inet6Address = nil
	}

	device, err := tun.New(tunOptions)
	if err != nil {
		return err
	}
	lu.SetDevice(device)

	stack, err := stack.New(&stack.Options{
		Tun:     device,
		Handler: h,
		Stack:   stack.TunGVisor,
	})
	if err != nil {
		return err
	}
	log.Debug("Starting stack..")
	err = stack.Start(context.Background())
	if err != nil {
		return err
	}

	lu.SetStack(stack)
	log.Debug("Luma successfully started")
	return nil
}

func (lu *Luma) SetDnsAdds(dnsAdds []netip.AddrPort) {
	lu.mu.Lock()
	lu.dnsAdds = dnsAdds
	lu.mu.Unlock()
}

func (lu *Luma) SetDevice(d tun.Tun) {
	lu.mu.Lock()
	lu.device = d
	lu.mu.Unlock()
}

func (lu *Luma) SetStack(s stack.Stack) {
	lu.mu.Lock()
	lu.stack = s
	lu.mu.Unlock()
}

/* NewConnection handles new TCP connections
func (lu *Luma) NewConnection(ctx context.Context, c net.Conn, m M.Metadata) error {
	//log.Debugf("New TCP connection, metadata is %s", c.Metadata().FiveTuple())

	if deadline.NeedAdditionalReadDeadline(c) {
		c = conn.NewDeadlineConn(c) // conn from sing should check NeedAdditionalReadDeadline
	}

	cMetadata := &metadata.Metadata{
		Network: metadata.TCP,
	}
	inbound.WithOptions(cMetadata, inbound.WithDstAddr(m.Destination), inbound.WithSrcAddr(m.Source), inbound.WithInAddr(c.LocalAddr()))

	lu.tunnel.HandleTCP(adapter.NewTCPConn(c, cMetadata))
	return nil
}

// NewConnection handles new UDP packets
func (lu *Luma) NewPacketConnection(ctx context.Context, conn network.PacketConn, m M.Metadata) error {
	defer func() { _ = conn.Close() }()
	//log.Debugf("New UDP connection, metadata is %s", c.Metadata().FiveTuple())
	mutex := sync.Mutex{}
	conn2 := bufio.NewNetPacketConn(conn)
	defer func() {
		mutex.Lock()
		defer mutex.Unlock()
		conn2 = nil
	}()
	rwOptions := network.ReadWaitOptions{}
	readWaiter, isReadWaiter := bufio.CreatePacketReadWaiter(conn)
	if isReadWaiter {
		readWaiter.InitializeReadWaiter(rwOptions)
	}
	for {
		var (
			buff *pool.Buffer
			dest M.Socksaddr
			err  error
		)
		if isReadWaiter {
			buff, dest, err = readWaiter.WaitReadPacket()
		} else {
			buff = rwOptions.NewPacketBuffer()
			dest, err = conn.ReadPacket(buff)
			if buff != nil {
				rwOptions.PostReturn(buff)
			}
		}
		if err != nil {
			buff.Release()
			if ShouldIgnorePacketError(err) {
				break
			}
			return err
		}

		cPacket := &packet{
			conn:  &conn2,
			mutex: &mutex,
			rAddr: m.Source.UDPAddr(),
			lAddr: conn.LocalAddr(),
			buff:  buff,
		}
		cMetadata := &metadata.Metadata{
			Network: metadata.UDP,
		}

		inbound.WithOptions(cMetadata, inbound.WithDstAddr(dest), inbound.WithSrcAddr(m.Source), inbound.WithInAddr(conn.LocalAddr()))
		lu.tunnel.HandleUDP(adapter.NewPacketAdapter(cPacket, cMetadata))
	}
	return nil
}*/

// Stop stops running the Luma engine
func (lu *Luma) Stop() {
	log.Debug("Stopping luma..")
	if lu.device != nil {
		lu.device.Close()
	}
	if lu.stack != nil {
		lu.stack.Stop()
	}
	if lu.socksServer != nil {
		lu.socksServer.Stop()
	}
}

func (lu *Luma) flushDefaultInterface() {
	log.Debug("Flushing default interface")
	targetInterface := dialer.DefaultInterface.Load()
	for _, destination := range []netip.Addr{netip.IPv4Unspecified(), netip.IPv6Unspecified(), netip.MustParseAddr("1.1.1.1")} {
		autoDetectInterfaceName := "en0"
		if autoDetectInterfaceName == lu.tunName {
			log.Warnf("[TUN] Auto detect interface by %s get same name with tun", destination.String())
		} else if autoDetectInterfaceName == "" || autoDetectInterfaceName == "<nil>" {
			log.Warnf("[TUN] Auto detect interface by %s get empty name.", destination.String())
		} else {
			targetInterface = autoDetectInterfaceName
			if old := dialer.DefaultInterface.Load(); old != targetInterface {
				log.Warnf("[TUN] default interface changed by monitor, %s => %s", old, targetInterface)

				dialer.DefaultInterface.Store(targetInterface)
				iface.FlushCache()
			}
			return
		}
	}
}

func ShouldIgnorePacketError(err error) bool {
	return false
}
