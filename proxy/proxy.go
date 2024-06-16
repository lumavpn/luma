package proxy

import (
	"context"
	"net"

	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/protos"
)

type Proxy interface {
	// Name returns the name of this proxy
	Name() string
	// Addr is the address of the proxy
	Addr() string
	// Protocol is the protocol of the proxy
	Protocol() protos.Protocol
	// SupportUDP returns whether or not the proxy supports UDP
	SupportUDP() bool
	// DialContext connects to the address on the network specified by Metadata
	DialContext(ctx context.Context, metadata *M.Metadata) (net.Conn, error)
}
