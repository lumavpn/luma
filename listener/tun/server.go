package tun

import (
	"context"
	"fmt"
	"net/netip"
	"runtime"
	"strconv"
	"strings"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/listener/mux"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy/inbound"
	"github.com/lumavpn/luma/proxy/protos"
	"github.com/lumavpn/luma/stack"
	"github.com/lumavpn/luma/util"
)

type Listener struct {
	closed  bool
	config  *config.Tun
	handler *ListenerHandler
	tunName string
	addrStr string

	tunIf stack.Tun
	stack stack.Stack
}

func checkTunName(tunName string) (ok bool) {
	if runtime.GOOS == "darwin" {
		if len(tunName) <= 4 {
			return false
		}
		if tunName[:4] != "utun" {
			return false
		}
		if _, parseErr := strconv.ParseInt(tunName[4:], 10, 16); parseErr != nil {
			return false
		}
	}
	return true
}

// New creates a new instance of Listener with the given config and options
func New(cfg *config.Tun, interfaceName string, tunnel adapter.TransportHandler, options ...inbound.Option) (*Listener, error) {
	if len(options) == 0 {
		options = []inbound.Option{
			inbound.WithInName("default-tun"),
		}
	}
	tunName := cfg.Device
	if tunName == "" || !checkTunName(tunName) {
		tunName = util.CalculateInterfaceName(interfaceName)
		log.Debugf("Setting tun device name to %s", tunName)
		cfg.Device = tunName
	}
	tunMTU := cfg.MTU
	if tunMTU == 0 {
		tunMTU = 9000
	}
	var dnsAdds []netip.AddrPort
	for _, d := range cfg.DNSHijack {
		if _, after, ok := strings.Cut(d, "://"); ok {
			d = after
		}
		d = strings.Replace(d, "any", "0.0.0.0", 1)
		addrPort, err := netip.ParseAddrPort(d)
		if err != nil {
			return nil, fmt.Errorf("parse dns-hijack url error: %w", err)
		}

		dnsAdds = append(dnsAdds, addrPort)
	}

	var dnsServerIp []string
	for _, a := range cfg.Inet4Address {
		addrPort := netip.AddrPortFrom(a.Addr().Next(), 53)
		dnsServerIp = append(dnsServerIp, a.Addr().Next().String())
		dnsAdds = append(dnsAdds, addrPort)
	}
	for _, a := range cfg.Inet6Address {
		addrPort := netip.AddrPortFrom(a.Addr().Next(), 53)
		dnsServerIp = append(dnsServerIp, a.Addr().Next().String())
		dnsAdds = append(dnsAdds, addrPort)
	}

	h, err := mux.NewListenerHandler(mux.ListenerConfig{
		Tunnel:  tunnel,
		Type:    protos.Protocol_TUN,
		Options: options,
	})
	if err != nil {
		return nil, err
	}
	handler := &ListenerHandler{
		ListenerHandler: h,
		DnsAdds:         dnsAdds,
	}
	l := &Listener{
		closed:  false,
		config:  cfg,
		handler: handler,
		tunName: tunName,
	}

	tunOptions := stack.Options{
		Name: tunName,
		MTU:  tunMTU,
	}
	tunIf, err := tunNew(tunOptions)
	if err != nil {
		err = fmt.Errorf("configure tun interface: %v", err)
		return nil, err
	}

	stackOptions := &stack.Config{
		Tun:        tunIf,
		TunOptions: tunOptions,
		Handler:    h,
	}

	l.tunIf = tunIf

	log.Debug("Creating new stack..")
	tunStack, err := stack.NewStack(stackOptions)
	if err != nil {
		return nil, err
	}
	log.Debug("Starting stack..")
	err = tunStack.Start(context.Background())
	if err != nil {
		return nil, err
	}
	l.stack = tunStack
	return l, nil
}

func (l *Listener) Close() error {
	l.closed = true
	return util.Close(
		l.stack,
		l.tunIf,
	)
}

func (l *Listener) Config() *config.Tun {
	return l.config
}

func (l *Listener) Address() string {
	return l.addrStr
}
