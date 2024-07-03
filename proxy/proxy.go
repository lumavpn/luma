package proxy

import (
	"context"
	"net"
	"net/netip"
	"time"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/dialer"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/util"
)

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
	ListenPacket(ctx context.Context, network, address string, rAddrPort netip.AddrPort) (net.PacketConn, error)
}

type ProxyAdapter interface {
	// Name returns the name of this proxy
	Name() string
	// Addr is the address of the proxy
	Addr() string
	// Proto is the protocol of the proxy
	Proto() proto.Proto
	// SupportUDP returns whether or not the proxy supports UDP
	SupportUDP() bool
	SupportXUDP() bool
	SupportTFO() bool
	MarshalJSON() ([]byte, error)

	DialContext(context.Context, *M.Metadata, ...dialer.Option) (Conn, error)
	ListenPacketContext(context.Context, *M.Metadata, ...dialer.Option) (PacketConn, error)

	// SupportUOT return UDP over TCP support
	SupportUOT() bool
	SupportWithDialer() M.Network

	DialOptions(opts ...dialer.Option) []dialer.Option
	DialContextWithDialer(ctx context.Context, dialer Dialer, metadata *M.Metadata) (Conn, error)
	ListenPacketWithDialer(ctx context.Context, dialer Dialer, metadata *M.Metadata) (PacketConn, error)

	Unwrap(metadata *M.Metadata, touch bool) Proxy

	IsL3Protocol(metadata *M.Metadata) bool
}

type Proxy interface {
	ProxyAdapter
	AliveForTestUrl(url string) bool
	ExtraDelayHistories() map[string]ProxyState
	LastDelayForTestUrl(url string) uint16
	URLTest(ctx context.Context, url string, expectedStatus util.IntRanges[uint16]) (uint16, error)
}

type ProxyState struct {
	Alive   bool           `json:"alive"`
	History []DelayHistory `json:"history"`
}

type DelayHistory struct {
	Time  time.Time `json:"time"`
	Delay uint16    `json:"delay"`
}

type DelayHistoryStoreType int

type WriteBackProxy interface {
	adapter.WriteBack
	UpdateWriteBack(wb adapter.WriteBack)
}
