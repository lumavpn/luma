package stack

import (
	"context"
	"fmt"
	"net"

	"github.com/lumavpn/luma/metadata"
)

type Stack interface {
	Start(context.Context) error
	Stop() error
}

type Options struct {
	Handler Handler
	Stack   StackType
}

type TCPConnectionHandler interface {
	NewConnection(ctx context.Context, conn net.Conn, m *metadata.Metadata) error
}

type UDPConnectionHandler interface {
	NewPacketConnection(ctx context.Context, conn net.PacketConn, m *metadata.Metadata) error
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
