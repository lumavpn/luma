package luma

import (
	"fmt"

	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/local"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy"
)

type configResult struct {
	locals  map[string]local.LocalServer
	proxies map[string]proxy.Proxy
}

// parseConfig is used to parse the general configuration used by Luma
func (lu *Luma) parseConfig(cfg *config.Config) (*configResult, error) {
	proxies, err := parseProxies(cfg)
	if err != nil {
		return nil, err
	}

	log.Debugf("Have %d proxies", len(proxies))

	localServers, err := parseLocal(cfg)
	if err != nil {
		return nil, err
	}

	log.Debugf("Have %d local servers", len(localServers))

	return &configResult{
		proxies: proxies,
		locals:  localServers,
	}, nil
}

// parseProxies returns a map of proxies that are present in the config
func parseProxies(cfg *config.Config) (map[string]proxy.Proxy, error) {
	proxies := make(map[string]proxy.Proxy)
	return proxies, nil
}

// parseLocal returns a map of local proxy servers that are currently running
func parseLocal(cfg *config.Config) (map[string]local.LocalServer, error) {
	servers := make(map[string]local.LocalServer)
	for index, mapping := range cfg.Locals {
		server, err := local.ParseLocal(mapping)
		if err != nil {
			return nil, fmt.Errorf("parse local server %d: %w", index, err)
		} else if _, exist := mapping[server.Name()]; exist {
			return nil, fmt.Errorf("server %s is the duplicate name", server.Name())
		}
		servers[server.Name()] = server
	}
	return servers, nil
}
