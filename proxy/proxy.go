package proxy

import (
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/proto"
)

type Proxy interface {
	// Name returns the name of this proxy
	Name() string
	// Addr is the address of the proxy
	Addr() string
	// Protocol is the protocol of the proxy
	Protocol() proto.Protocol
	// SupportUDP returns whether or not the proxy supports UDP
	SupportUDP() bool
	Unwrap(*metadata.Metadata, bool) Proxy
}
