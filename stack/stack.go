package stack

import (
	"context"
	"fmt"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/stack/tun"
)

type Stack interface {
	Start(context.Context) error
	Stop() error
}

type Options struct {
	Handler Handler
	Stack   StackType

	Device tun.Device

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
