package listener

import "github.com/lumavpn/luma/adapter"

type InboundListener interface {
	Name() string
	Listen(tunnel adapter.TransportHandler) error
	Close() error
	Address() string
	RawAddress() string
}

type Listener interface {
	RawAddress() string
	Address() string
	Close() error
}
