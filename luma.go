package luma

import (
	"context"
	"net"
	"sync"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/stack"
	"github.com/lumavpn/luma/stack/tun"
	"github.com/lumavpn/luma/tunnel"
)

type Luma struct {
	// config is the configuration this instance of Luma is using
	config *config.Config
	// proxies is a map of proxies that Luma is configured to proxy traffic through
	proxies map[string]proxy.Proxy

	stack stack.Stack
	// Tunnel
	device tun.Device
	tunnel tunnel.Tunnel

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
		log.Infof("[DIALER] bind to interface: %s", cfg.Interface)
	}

	defaultProxy := proxy.NewDirect()
	proxy.SetDialer(defaultProxy)

	device, err := tun.New(tun.Options{
		Name: cfg.Device,
		MTU:  tunMTU,
	})
	if err != nil {
		return err
	}
	lu.SetDevice(device)

	stack, err := stack.New(&stack.Options{
		Device:  device,
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

func (lu *Luma) SetDevice(d tun.Device) {
	lu.mu.Lock()
	lu.device = d
	lu.mu.Unlock()
}

func (lu *Luma) SetStack(s stack.Stack) {
	lu.mu.Lock()
	lu.stack = s
	lu.mu.Unlock()
}

func (lu *Luma) NewConnection(ctx context.Context, conn adapter.TCPConn) error {
	log.Debug("New TCP connection")
	lu.tunnel.HandleTCP(conn)
	return nil
}

func (lu *Luma) NewPacketConnection(ctx context.Context, conn adapter.UDPConn) error {
	log.Debug("New UDP connection")
	lu.tunnel.HandleUDP(conn)
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

// startStack applies the given Config to the instance of Luma to complete setup
func (lu *Luma) startStack(cfg *config.Config) error {
	return nil
}
