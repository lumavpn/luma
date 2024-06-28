package tun

import (
	"io"
	"net/netip"

	"github.com/lumavpn/luma/common/network"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

const Driver = "tun"

type Tun interface {
	io.ReadWriter
	network.VectorisedWriter
	Close() error
}

type Options struct {
	FileDescriptor           int
	Name                     string
	Inet4Address             []netip.Prefix
	Inet6Address             []netip.Prefix
	MTU                      uint32
	GSO                      bool
	AutoRoute                bool
	StrictRoute              bool
	WireGuard                bool
	Inet4RouteAddress        []netip.Prefix
	Inet6RouteAddress        []netip.Prefix
	Inet4RouteExcludeAddress []netip.Prefix
	Inet6RouteExcludeAddress []netip.Prefix
}

type GVisorTun interface {
	Tun
	NewEndpoint() (stack.LinkEndpoint, error)
}
