package luma

import (
	"github.com/lumavpn/luma/local"
	"github.com/lumavpn/luma/log"
)

// startLocal iterates through local servers that Luma is configured with and starts them
func (lu *Luma) startLocal(newServers map[string]local.LocalServer, dropOld bool) {
	lu.mu.Lock()
	defer lu.mu.Unlock()
	for name, newServer := range newServers {
		if oldServer, ok := lu.localServers[name]; ok {
			if !oldServer.Config().Equal(newServer.Config()) {
				_ = oldServer.Close()
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
	if dropOld {
		for name, oldServer := range lu.localServers {
			if _, ok := newServers[name]; !ok {
				_ = oldServer.Close()
				delete(lu.localServers, name)
			}
		}
	}
}
