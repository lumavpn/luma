package luma

import "github.com/lumavpn/luma/config"

type Luma struct {
	config *config.Config
}

// New creates a new instance of Luma
func New(cfg *config.Config) *Luma {
	return &Luma{
		config: cfg,
	}
}
