package proxy

import (
	"context"
	"time"

	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/protos"
)

type ProxyAdapter interface {
	// Name returns the name of this proxy
	Name() string
	// Addr is the address of the proxy
	Addr() string
	// Protocol is the protocol of the proxy
	Protocol() protos.Protocol
	// SupportUDP returns whether or not the proxy supports UDP
	SupportUDP() bool
	// DialContext connects to the address on the network specified by Metadata
	DialContext(ctx context.Context, metadata *metadata.Metadata, opts ...dialer.Option) (Conn, error)
	ListenPacketContext(context.Context, *metadata.Metadata, ...dialer.Option) (PacketConn, error)
}

type Proxy interface {
	ProxyAdapter
	AliveForTestUrl(url string) bool
}

type DelayHistory struct {
	Time  time.Time `json:"time"`
	Delay uint16    `json:"delay"`
}
