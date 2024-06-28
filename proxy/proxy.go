package proxy

import (
	"context"
	"net"
	"time"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/proto"
)

const (
	tcpConnectTimeout = 5 * time.Second
)

type Proxy interface {
	Dialer
	// Name returns the name of this proxy
	Name() string
	// Addr is the address of the proxy
	Addr() string
	// Proto is the protocol of the proxy
	Proto() proto.Proto
	// SupportUDP returns whether or not the proxy supports UDP
	SupportUDP() bool
}

type WriteBackProxy interface {
	adapter.WriteBack
	UpdateWriteBack(wb adapter.WriteBack)
}

var _defaultDialer Dialer = &Base{}

type Dialer interface {
	DialContext(context.Context, *metadata.Metadata) (net.Conn, error)
	DialUDP(*metadata.Metadata) (net.PacketConn, error)
	ListenPacketContext(context.Context, *metadata.Metadata) (PacketConn, error)
}

// SetDialer sets default Dialer.
func SetDialer(d Dialer) {
	_defaultDialer = d
}

// Dial uses default Dialer to dial TCP.
func Dial(m *metadata.Metadata) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), tcpConnectTimeout)
	defer cancel()
	return _defaultDialer.DialContext(ctx, m)
}

// DialContext uses default Dialer to dial TCP with context.
func DialContext(ctx context.Context, m *metadata.Metadata) (net.Conn, error) {
	return _defaultDialer.DialContext(ctx, m)
}

// DialUDP uses default Dialer to dial UDP.
func DialUDP(m *metadata.Metadata) (net.PacketConn, error) {
	return _defaultDialer.DialUDP(m)
}
