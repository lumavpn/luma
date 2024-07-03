package luma

import (
	"context"
	"errors"
	"net"
	"slices"
	"strconv"

	"github.com/lumavpn/luma/adapter"
	C "github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/component/ebpf"
	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/listener"
	"github.com/lumavpn/luma/listener/autoredir"
	"github.com/lumavpn/luma/listener/http"
	"github.com/lumavpn/luma/listener/mixed"
	"github.com/lumavpn/luma/listener/redir"
	"github.com/lumavpn/luma/listener/socks"
	"github.com/lumavpn/luma/listener/tproxy"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy/inbound"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/tunnel"
)

var (
	errZeroPort = errors.New("port is invalid")
)

type Ports struct {
	Port              int    `json:"port"`
	SocksPort         int    `json:"socks-port"`
	RedirPort         int    `json:"redir-port"`
	TProxyPort        int    `json:"tproxy-port"`
	MixedPort         int    `json:"mixed-port"`
	ShadowSocksConfig string `json:"ss-config"`
	VmessConfig       string `json:"vmess-config"`
}

// GetPorts return the ports of proxy servers
func (lu *Luma) GetPorts() *Ports {
	ports := &Ports{}
	if lu.httpListener != nil {
		_, portStr, _ := net.SplitHostPort(lu.httpListener.Address())
		port, _ := strconv.Atoi(portStr)
		ports.Port = port
	}

	if lu.socksListener != nil {
		_, portStr, _ := net.SplitHostPort(lu.socksListener.Address())
		port, _ := strconv.Atoi(portStr)
		ports.SocksPort = port
	}

	if lu.redirListener != nil {
		_, portStr, _ := net.SplitHostPort(lu.redirListener.Address())
		port, _ := strconv.Atoi(portStr)
		ports.RedirPort = port
	}

	if lu.tproxyListener != nil {
		_, portStr, _ := net.SplitHostPort(lu.tproxyListener.Address())
		port, _ := strconv.Atoi(portStr)
		ports.TProxyPort = port
	}

	if lu.mixedListener != nil {
		_, portStr, _ := net.SplitHostPort(lu.mixedListener.Address())
		port, _ := strconv.Atoi(portStr)
		ports.MixedPort = port
	}
	return ports
}

func (lu *Luma) recreateAutoRedir(ifaceNames []string, tunnel adapter.TransportHandler) error {
	lu.autoRedirMu.Lock()
	defer lu.autoRedirMu.Unlock()

	var err error
	defer func() {
		if err != nil {
			if lu.autoRedirListener != nil {
				_ = lu.autoRedirListener.Close()
				lu.autoRedirListener = nil
			}
			if lu.autoRedirProgram != nil {
				lu.autoRedirProgram.Close()
				lu.autoRedirProgram = nil
			}
			log.Errorf("Start auto redirect server error: %s", err.Error())
		}
	}()

	nicArr := ifaceNames
	slices.Sort(nicArr)
	nicArr = slices.Compact(nicArr)

	if lu.autoRedirListener != nil && lu.autoRedirProgram != nil {
		_ = lu.autoRedirListener.Close()
		lu.autoRedirProgram.Close()
		lu.autoRedirListener = nil
		lu.autoRedirProgram = nil
	}

	if len(nicArr) == 0 {
		return nil
	}

	defaultRouteInterfaceName, err := ebpf.GetAutoDetectInterface()
	if err != nil {
		return err
	}

	addr := generateAddress("*", C.TcpAutoRedirPort, true)

	if portIsZero(addr) {
		return nil
	}

	lu.autoRedirListener, err = autoredir.New(addr, tunnel)
	if err != nil {
		return err
	}

	lu.autoRedirProgram, err = ebpf.NewRedirEBpfProgram(nicArr, lu.autoRedirListener.TCPAddr().Port(),
		defaultRouteInterfaceName)
	if err != nil {
		return err
	}

	lu.autoRedirListener.SetLookupFunc(lu.autoRedirProgram.Lookup)

	log.Infof("Auto redirect proxy listening at: %s, attached tc ebpf program to interfaces %v",
		lu.autoRedirListener.Address(), lu.autoRedirProgram.RawNICs())
	return nil
}

func (lu *Luma) recreateRedir(ctx context.Context, cfg *config.Config, tunnel tunnel.Tunnel) error {
	lu.redirMu.Lock()
	defer lu.redirMu.Unlock()

	var err error
	defer func() {
		if err != nil {
			log.Errorf("Start Redir server error: %s", err.Error())
		}
	}()

	addr := generateAddress(cfg.BindAddress, cfg.RedirPort, cfg.AllowLan)

	if lu.redirListener != nil {
		if lu.redirListener.RawAddress() == addr {
			return nil
		}
		lu.redirListener.Close()
		lu.redirListener = nil
	}

	if lu.redirListener != nil {
		if lu.redirListener.RawAddress() == addr {
			return nil
		}
		lu.redirListener.Close()
		lu.redirListener = nil
	}

	if portIsZero(addr) {
		return nil
	}

	lu.redirListener, err = redir.New(addr, tunnel)
	if err != nil {
		return err
	}

	lu.redirUDPListener, err = tproxy.NewUDP(addr, tunnel)
	if err != nil {
		log.Warnf("Failed to start Redir UDP Listener: %s", err)
	}

	log.Infof("Redirect proxy listening at: %s", lu.redirListener.Address())
	return nil
}

func recreateListener(generateAddress func() string, setListener func(listener.Listener), l listener.Listener, proto proto.Proto,
	tunnel adapter.TransportHandler, fn func(string, adapter.TransportHandler, ...inbound.Addition) (listener.Listener, error)) error {
	addr := generateAddress()

	if l != nil {
		if l.RawAddress() == addr {
			return nil
		}
		l.Close()
		setListener(nil)
	}

	if portIsZero(addr) {
		return nil
	}
	result, err := fn(addr, tunnel)
	if err != nil {
		log.Errorf("Start %s server error: %s", proto.String(), err.Error())
		return err
	}
	setListener(result)
	log.Infof("%s proxy listening at: %s", proto.String(), l.Address())
	return nil
}

func (lu *Luma) recreateHTTP(ctx context.Context, cfg *config.Config, tunnel adapter.TransportHandler) error {
	var err error
	defer func() {
		if err != nil {
			log.Errorf("Start HTTP server error: %s", err.Error())
		}
	}()

	addr := generateAddress(cfg.BindAddress, cfg.Port, cfg.AllowLan)

	if lu.httpListener != nil {
		if lu.httpListener.RawAddress() == addr {
			return nil
		}
		lu.httpListener.Close()
		lu.httpListener = nil
	}

	if portIsZero(addr) {
		return nil
	}

	lu.httpListener, err = http.New(addr, tunnel)
	if err != nil {
		log.Errorf("Start HTTP server error: %s", err.Error())
		return err
	}

	log.Infof("HTTP proxy listening at: %s", lu.httpListener.Address())
	return nil
}

func (lu *Luma) recreateSocks(ctx context.Context, cfg *config.Config, tunnel adapter.TransportHandler) error {
	lu.socksMu.Lock()
	defer lu.socksMu.Unlock()
	socksListener := lu.socksListener
	socksUDPListener := lu.socksUDPListener

	var err error
	defer func() {
		if err != nil {
			log.Errorf("Start SOCKS server error: %s", err.Error())
		}
	}()
	addr := generateAddress(cfg.BindAddress, cfg.SocksPort, cfg.AllowLan)

	shouldTCPIgnore := false
	shouldUDPIgnore := false

	if socksListener != nil {
		if socksListener.RawAddress() != addr {
			socksListener.Close()
			lu.socksListener = nil
		} else {
			shouldTCPIgnore = true
		}
	}

	if socksUDPListener != nil {
		if socksUDPListener.RawAddress() != addr {
			socksUDPListener.Close()
			lu.socksUDPListener = nil
		} else {
			shouldUDPIgnore = true
		}
	}

	if shouldTCPIgnore && shouldUDPIgnore {
		return errors.New("ignoring both tcp and udp, not starting server")
	}

	if portIsZero(addr) {
		return nil
	}
	log.Debug("Starting socks proxy server")
	tcpListener, err := socks.New(addr, tunnel)
	if err != nil {
		return err
	}

	udpListener, err := socks.NewUDP(addr, tunnel)
	if err != nil {
		tcpListener.Close()
		return err
	}

	lu.socksListener = tcpListener
	lu.socksUDPListener = udpListener

	log.Debugf("SOCKS proxy listening at: %s", tcpListener.Address())
	return nil
}

func (lu *Luma) recreateMixed(cfg *config.Config, tunnel adapter.TransportHandler) error {
	lu.mixedMu.Lock()
	defer lu.mixedMu.Unlock()

	var err error
	defer func() {
		if err != nil {
			log.Errorf("Start Mixed(http+socks) server error: %s", err.Error())
		}
	}()

	addr := generateAddress(cfg.BindAddress, cfg.MixedPort, cfg.AllowLan)

	shouldTCPIgnore := false
	shouldUDPIgnore := false

	if lu.mixedListener != nil {
		if lu.mixedListener.RawAddress() != addr {
			lu.mixedListener.Close()
			lu.mixedListener = nil
		} else {
			shouldTCPIgnore = true
		}
	}
	if lu.mixedUDPLister != nil {
		if lu.mixedUDPLister.RawAddress() != addr {
			lu.mixedUDPLister.Close()
			lu.mixedUDPLister = nil
		} else {
			shouldUDPIgnore = true
		}
	}

	if shouldTCPIgnore && shouldUDPIgnore {
		return errors.New("ignoring both tcp and udp, not starting server")
	}

	if portIsZero(addr) {
		return nil
	}

	lu.mixedListener, err = mixed.New(addr, tunnel)
	if err != nil {
		return err
	}

	lu.mixedUDPLister, err = socks.NewUDP(addr, tunnel)
	if err != nil {
		lu.mixedListener.Close()
		return err
	}

	log.Infof("Mixed(http+socks) proxy listening at: %s", lu.mixedListener.Address())
	return nil
}

func (lu *Luma) recreateTProxy(cfg *config.Config, tunnel tunnel.Tunnel) error {
	lu.tproxyMu.Lock()
	defer lu.tproxyMu.Unlock()

	var err error
	defer func() {
		if err != nil {
			log.Errorf("Start TProxy server error: %s", err.Error())
		}
	}()

	addr := generateAddress(cfg.BindAddress, cfg.TProxyPort, cfg.AllowLan)

	if lu.tproxyListener != nil {
		if lu.tproxyListener.RawAddress() == addr {
			return nil
		}
		lu.tproxyListener.Close()
		lu.tproxyListener = nil
	}

	if lu.tproxyUDPListener != nil {
		if lu.tproxyUDPListener.RawAddress() == addr {
			return nil
		}
		lu.tproxyUDPListener.Close()
		lu.tproxyUDPListener = nil
	}

	if portIsZero(addr) {
		return nil
	}

	lu.tproxyListener, err = tproxy.New(addr, tunnel)
	if err != nil {
		return err
	}

	lu.tproxyUDPListener, err = tproxy.NewUDP(addr, tunnel)
	if err != nil {
		log.Warnf("Failed to start TProxy UDP Listener: %s", err)
	}

	log.Infof("TProxy server listening at: %s", lu.tproxyListener.Address())
	return nil
}
