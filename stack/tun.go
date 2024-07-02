package stack

import (
	"context"
	"io"
	"net/netip"

	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/common/ranges"
)

type Handler interface {
	N.TCPConnectionHandler
	N.UDPConnectionHandler
	NewError(context.Context, error)
}

type Tun interface {
	io.ReadWriter
	N.VectorisedWriter
	Close() error
}

type Options struct {
	Name                     string
	Inet4Address             []netip.Prefix
	Inet6Address             []netip.Prefix
	MTU                      uint32
	GSO                      bool
	AutoRoute                bool
	StrictRoute              bool
	Inet4RouteAddress        []netip.Prefix
	Inet6RouteAddress        []netip.Prefix
	Inet4RouteExcludeAddress []netip.Prefix
	Inet6RouteExcludeAddress []netip.Prefix
	IncludeInterface         []string
	ExcludeInterface         []string
	IncludeUID               []ranges.Range[uint32]
	ExcludeUID               []ranges.Range[uint32]
	IncludeAndroidUser       []int
	IncludePackage           []string
	ExcludePackage           []string
	TableIndex               int
	FileDescriptor           int
}
