package luma

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/common/bufio"
	"github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/pool"
	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/inbound"
	"github.com/lumavpn/luma/stack"
	"github.com/lumavpn/luma/stack/tun"
	"github.com/lumavpn/luma/tunnel"
	"github.com/lumavpn/luma/util"
)

type Luma struct {
	// config is the configuration this instance of Luma is using
	config *config.Config
	// proxies is a map of proxies that Luma is configured to proxy traffic through
	proxies map[string]proxy.Proxy

	stack stack.Stack
	// Tunnel
	device  tun.Tun
	dnsAdds []netip.AddrPort
	tunName string
	tunnel  tunnel.Tunnel

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
	cfg := lu.config
	tunMTU := cfg.MTU
	if tunMTU == 0 {
		tunMTU = 9000
	}

	if cfg.Interface != "" {
		iface, err := net.InterfaceByName(cfg.Interface)
		if err != nil {
			return err
		}
		dialer.DefaultInterfaceName.Store(iface.Name)
		dialer.DefaultInterfaceIndex.Store(int32(iface.Index))
		log.Infof("bind to interface: %s", cfg.Interface)
	}

	defaultProxy := proxy.NewDirect()
	proxy.SetDialer(defaultProxy)

	tunName := cfg.Device
	if tunName == "" || !checkTunName(tunName) {
		tunName = util.CalculateInterfaceName("Luma")
		log.Debugf("Setting tun device name to %s", tunName)
		cfg.Device = tunName
	}

	var dnsAdds []netip.AddrPort

	for _, d := range cfg.DNSHijack {
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

	device, err := tun.New(tun.Options{
		AutoRoute: true,
		Name:      cfg.Device,
		MTU:       tunMTU,
		WireGuard: true,
	})
	if err != nil {
		return err
	}
	lu.SetDevice(device)

	stack, err := stack.New(&stack.Options{
		Tun:     device,
		Handler: lu,
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

// NewConnection handles new TCP connections
func (lu *Luma) NewConnection(ctx context.Context, c adapter.TCPConn) error {
	/*tcpConn := c.Conn()
	lAddr := tcpConn.RemoteAddr()
	rAddr := tcpConn.LocalAddr()
	if lAddr == nil || rAddr == nil {
		log.Debug("No left or right address")
		return nil
	}
	source := M.ParseSocksAddrFromNet(lAddr)
	destination := M.ParseSocksAddrFromNet(rAddr)
	v := map[string]any{
		"source":      source,
		"destination": destination,
	}*/
	log.Debugf("New TCP connection, metadata is %s", c.Metadata().FiveTuple())
	lu.tunnel.HandleTCP(c)
	return nil
}

// NewConnection handles new UDP packets
func (lu *Luma) NewPacketConnection(ctx context.Context, c adapter.UDPConn) error {
	log.Debugf("New UDP connection, metadata is %s", c.Metadata().FiveTuple())
	mutex := sync.Mutex{}
	conn := c.Conn()
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
			dest metadata.Socksaddr
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

		m := c.Metadata()
		inbound.WithOptions(m, inbound.WithDstAddr(dest))

		cPacket := &packet{
			conn:  &conn2,
			mutex: &mutex,
			rAddr: m.Source.UDPAddr(),
			lAddr: conn.LocalAddr(),
			buff:  buff,
		}
		lu.tunnel.HandleUDP(adapter.NewPacketAdapter(cPacket, m))
	}
	return nil
}

// Stop stops running the Luma engine
func (lu *Luma) Stop() {
	log.Debug("Stopping luma..")
	if lu.device != nil {
		lu.device.Close()
	}
	if lu.stack != nil {
		lu.stack.Stop()
	}
}

func (lu *Luma) FlushDefaultInterface() {
	targetInterface := dialer.DefaultInterfaceName.Load()
	for _, destination := range []netip.Addr{netip.IPv4Unspecified(), netip.IPv6Unspecified(), netip.MustParseAddr("1.1.1.1")} {
		autoDetectInterfaceName := "en0"
		if autoDetectInterfaceName == lu.tunName {
			log.Warnf("[TUN] Auto detect interface by %s get same name with tun", destination.String())
		} else if autoDetectInterfaceName == "" || autoDetectInterfaceName == "<nil>" {
			log.Warnf("[TUN] Auto detect interface by %s get empty name.", destination.String())
		} else {
			targetInterface = autoDetectInterfaceName
			if old := dialer.DefaultInterfaceName.Load(); old != targetInterface {
				log.Warnf("[TUN] default interface changed by monitor, %s => %s", old, targetInterface)

				dialer.DefaultInterfaceName.Store(targetInterface)
				//iface.FlushCache()
			}
			return
		}
	}
}

func ShouldIgnorePacketError(err error) bool {
	return false
}
