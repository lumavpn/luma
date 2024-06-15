package test

import (
	"context"
	"testing"

	"github.com/lumavpn/luma"
	"github.com/lumavpn/luma/config"
	"github.com/stretchr/testify/require"
)

func TestLuma(t *testing.T) {
	ctx := context.Background()
	basic := `
loglevel: debug
`
	cfg, err := config.ParseBytes([]byte(basic))
	require.NoError(t, err)
	lu := luma.New(cfg)
	err = lu.Start(ctx)
	require.NoError(t, err)

}
