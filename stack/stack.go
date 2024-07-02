package stack

import (
	"context"
	"fmt"
)

type Stack interface {
	Start(context.Context) error
	Close() error
}

type Config struct {
	Context                context.Context
	EndpointIndependentNat bool
	Stack                  StackType
	Tun                    Tun
	TunOptions             Options
	UDPTimeout             int64
	Handler                Handler
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
