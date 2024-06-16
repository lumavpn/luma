package inbound

import "github.com/lumavpn/luma/adapter"

type InboundListener interface {
	Name() string
	Listen(tunnel adapter.TransportHandler) error
	Close() error
	Address() string
	RawAddress() string
	Config() InboundConfig
}

type InboundConfig interface {
	Name() string
	Equal(config InboundConfig) bool
}
