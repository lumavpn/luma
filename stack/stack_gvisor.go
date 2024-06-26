//go:build with_gvisor

package stack

import (
	"context"

	"github.com/lumavpn/luma/log"

	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

type gVisor struct {
	stack *stack.Stack
}

func NewGVisor(
	options *Options,
) (Stack, error) {
	log.Debug("Creating new gVisor stack")
	return &gVisor{}, nil
}

func (t *gVisor) Start(ctx context.Context) error {
	return nil
}

func (t *gVisor) Stop() error {
	return nil
}
