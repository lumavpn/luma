package luma

import (
	"context"

	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/log"
)

type Luma struct {
	config *config.Config
}

// New creates a new instance of Luma
func New(cfg *config.Config) *Luma {
	return &Luma{
		config: cfg,
	}
}

// Start starts the default engine running Luma
// If there is any issue with the setup process, an error is returned
func (lu *Luma) Start(ctx context.Context) error {
	log.Debug("Starting new instance")
	return nil
}

// Stop stops running the Luma engine. It returns an error if there was an issue doing so.
func (lu *Luma) Stop() error {
	return nil
}
