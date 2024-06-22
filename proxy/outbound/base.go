package outbound

import (
	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/proxy/protos"
)

type Base struct {
	name          string
	addr          string
	at            protos.AdapterType
	interfaceName string
	routingMark   int
	proto         protos.Protocol
	udp           bool
}

type BasicOptions struct {
	Name        string
	Addr        string
	Protocol    protos.Protocol
	Type        protos.AdapterType
	UDP         bool
	Interface   string
	RoutingMark int
}

// Addr returns the address of the proxy
func (b *Base) Addr() string {
	return b.addr
}

// Name returns the name of the proxy
func (b *Base) Name() string {
	return b.name
}

// AdapterType returns the adapter type the proxy is configured with
func (b *Base) AdapterType() protos.AdapterType {
	return b.at
}

// Protocol returns the protocol of the proxy
func (b *Base) Protocol() protos.Protocol {
	return b.proto
}

// SupportUDP returns whether or not the proxy supports UDP
func (b *Base) SupportUDP() bool {
	return b.udp
}

// DialOptions return []dialer.Option from struct
func (b *Base) DialOptions(opts ...dialer.Option) []dialer.Option {
	if b.interfaceName != "" {
		opts = append(opts, dialer.WithInterface(b.interfaceName))
	}
	if b.routingMark != 0 {
		opts = append(opts, dialer.WithRoutingMark(b.routingMark))
	}

	return opts
}
