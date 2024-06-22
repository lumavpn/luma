package luma

import (
	"fmt"

	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/listener/socks"
	"github.com/lumavpn/luma/listener/tun"
	"github.com/lumavpn/luma/log"
)

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

// setupTun parses the TUN configuration and creates a new tun.Listener if enabled that
// intercepts all traffic
func (lu *Luma) setupTun(cfg *config.Config) error {
	tunConfig, err := parseTun(cfg, lu.tunnel)
	if err != nil {
		return err
	}
	if !tunConfig.Enable {
		// tunnel is disabled
		return nil
	}
	listener, err := tun.New(tunConfig, lu.tunnel)
	if err != nil {
		return err
	}
	lu.mu.Lock()
	lu.tunListener = listener
	lu.mu.Unlock()
	return nil
}
