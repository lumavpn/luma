package outbound

import "github.com/lumavpn/luma/proxy/protos"

type Base struct {
	name  string
	addr  string
	at    protos.AdapterType
	proto protos.Protocol
	udp   bool
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