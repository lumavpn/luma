package proxy

import (
	"context"
	"time"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/protos"
)

type Dialer interface {
	// DialContext connects to the address on the network specified by Metadata
	DialContext(ctx context.Context, metadata *metadata.Metadata, opts ...dialer.Option) (Conn, error)
	ListenPacketContext(context.Context, *metadata.Metadata, ...dialer.Option) (PacketConn, error)
}

type ProxyAdapter interface {
	Dialer
	// Name returns the name of this proxy
	Name() string
	// Addr is the address of the proxy
	Addr() string
	// AdapterType is the adapter type of the proxy
	AdapterType() protos.AdapterType
	// Protocol is the protocol of the proxy
	Protocol() protos.Protocol
	// SupportUDP returns whether or not the proxy supports UDP
	SupportUDP() bool
	Unwrap(*metadata.Metadata, bool) Proxy
}

type Proxy interface {
	ProxyAdapter
	AliveForTestUrl(url string) bool
}

type DelayHistory struct {
	Time  time.Time `json:"time"`
	Delay uint16    `json:"delay"`
}

type WriteBackProxy interface {
	adapter.WriteBack
	UpdateWriteBack(wb adapter.WriteBack)
}
