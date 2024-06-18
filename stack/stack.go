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
	Handler    Handler
	Stack      StackType
	Tun        Tun
	TunOptions Options
}

// NewStack creates a new instance of Stack with the given options
func NewStack(cfg *Config) (Stack, error) {
	return nil, fmt.Errorf("unknown stack: %s", cfg.Stack)
}
