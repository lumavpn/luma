//go:build with_notray

package main

import (
	"context"
	"os"

	luma "github.com/lumavpn/luma"
	"github.com/lumavpn/luma/log"
)

func Start(ctx context.Context, lu *luma.Luma) {
	if err := lu.Start(ctx); err != nil {
		log.Errorf("unable to create luma: %v", err)
		os.Exit(1)
	}
}
