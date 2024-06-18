//go:build with_gvisor

package stack

import (
	"context"
	"errors"

	"github.com/lumavpn/luma/log"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

type gVisor struct {
	config   *Config
	handler  Handler
	stack    *stack.Stack
	endpoint stack.LinkEndpoint
	tun      GVisorTun
}

type GVisorTun interface {
	Tun
	NewEndpoint() (stack.LinkEndpoint, error)
}

func NewGVisor(
	cfg *Config,
) (Stack, error) {
	gTun, isGTun := cfg.Tun.(GVisorTun)
	if !isGTun {
		return nil, errors.New("gVisor stack is unsupported on current platform")
	}
	log.Debug("Creating new gVisor stack")
	return &gVisor{
		config:  cfg,
		handler: cfg.Handler,
		tun:     gTun,
	}, nil
}

func (t *gVisor) Start(ctx context.Context) error {
	return nil
}

func (t *gVisor) Close() error {
	t.endpoint.Attach(nil)
	t.stack.Close()
	for _, endpoint := range t.stack.CleanupEndpoints() {
		endpoint.Abort()
	}
	return nil
}
