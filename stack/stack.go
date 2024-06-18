package stack

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
)

type Stack interface {
	Start(context.Context) error
	Close() error
}

type Config struct {
	EndpointIndependentNat bool
	Handler                Handler
	Stack                  StackType
	Tun                    Tun
	TunOptions             Options
	UDPTimeout             int64
}

// NewStack creates a new instance of Stack with the given options
func NewStack(cfg *Config) (Stack, error) {
	switch cfg.Stack {
	case TunGVisor:
		return NewGVisor(cfg)
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
