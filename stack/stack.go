package stack

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"

	"github.com/lumavpn/luma/common/control"
)

type Stack interface {
	Start(context.Context) error
	Close() error
}

type Config struct {
	Context                context.Context
	Stack                  StackType
	Tun                    Tun
	TunOptions             Options
	EndpointIndependentNat bool
	UDPTimeout             int64
	Handler                Handler
	ForwarderBindInterface bool
	IncludeAllNetworks     bool
	InterfaceFinder        control.InterfaceFinder
}

// NewStack creates a new instance of Stack with the given options
func NewStack(cfg *Config) (Stack, error) {
	switch cfg.Stack {
	case TunGVisor:
		return NewGVisor(cfg)
	case TunMixed:
		if cfg.IncludeAllNetworks {
			return nil, ErrIncludeAllNetworks
		}
		return NewMixed(cfg)
	case TunSystem:
		if cfg.IncludeAllNetworks {
			return nil, ErrIncludeAllNetworks
		}
		return NewSystem(cfg)
	default:
		return nil, fmt.Errorf("unknown stack: %s", cfg.Stack)
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
