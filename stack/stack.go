package stack

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/stack/tun"
)

type Stack interface {
	Start(context.Context) error
	Stop() error
}

type Options struct {
	Handler      Handler
	Stack        StackType
	Tun          tun.Tun
	Device       tun.Device
	Inet4Address []netip.Prefix
	Inet6Address []netip.Prefix
	UDPTimeout   int64
	//TunOptions tun.Options
}

type TCPConnectionHandler interface {
	NewConnection(ctx context.Context, conn adapter.TCPConn) error
}

type UDPConnectionHandler interface {
	NewPacketConnection(ctx context.Context, conn adapter.UDPConn) error
}

type Handler interface {
	TCPConnectionHandler
	UDPConnectionHandler
}

// New creates a new instance of Stack with the given options
func New(options *Options) (Stack, error) {
	switch options.Stack {
	case TunGVisor:
		return NewGVisor(options)
	case TunSystem:
		return NewSystem(options)
	default:
		return nil, fmt.Errorf("unknown stack: %s", options.Stack)
	}
}

func BroadcastAddr(inet4Address []netip.Prefix) netip.Addr {
	if len(inet4Address) == 0 {
		return netip.Addr{}
	}
	prefix := inet4Address[0]
	var broadcastAddr [4]byte
	binary.BigEndian.PutUint32(broadcastAddr[:], binary.BigEndian.Uint32(prefix.Masked().Addr().AsSlice())|^binary.BigEndian.Uint32(net.CIDRMask(prefix.Bits(), 32)))
	return netip.AddrFrom4(broadcastAddr)
}
