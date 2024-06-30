package luma

import (
	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/local"
	"github.com/lumavpn/luma/log"
)

// startLocal iterates through local servers that Luma is configured with and starts them
func (lu *Luma) startLocal(newServers map[string]local.LocalServer, stopOld bool) {
	lu.mu.Lock()
	defer lu.mu.Unlock()
	for name, newServer := range newServers {
		if oldServer, ok := lu.localServers[name]; ok {
			if !oldServer.Config().Equal(newServer.Config()) {
				_ = oldServer.Stop()
			} else {
				continue
			}
		}
		if err := newServer.Start(lu.tunnel); err != nil {
			log.Errorf("Local server %s start err: %s", name, err.Error())
			continue
		}
		lu.localServers[name] = newServer
	}
	if stopOld {
		for name, oldServer := range lu.localServers {
			if _, ok := newServers[name]; !ok {
				_ = oldServer.Stop()
				delete(lu.localServers, name)
			}
		}
	}
}

func (lu *Luma) localSocksServer(cfg *config.Config) error {
	addr := generateAddress(cfg.BindAll, cfg.SocksPort)
	if portIsZero(addr) {
		return nil
	}
	socksServer, err := local.NewSocks(&local.SocksOption{
		Addr: addr,
		UDP:  true,
	})
	if err != nil {
		return err
	}
	go socksServer.Start(lu.tunnel)
	lu.mu.Lock()
	lu.socksServer = socksServer
	lu.mu.Unlock()
	return nil
}
