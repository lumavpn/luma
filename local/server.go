package local

import "github.com/lumavpn/luma/adapter"

type LocalServer interface {
	Name() string
	Start(tunnel adapter.TransportHandler) error
	Close() error
	Address() string
	RawAddress() string
	Config() LocalConfig
}

type LocalConfig interface {
	Name() string
	Equal(config LocalConfig) bool
}
