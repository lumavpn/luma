package adapter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseProxy(t *testing.T) {
	_, err := ParseProxy(map[string]any{
		"type": "unknown",
		"name": "UNKNOWN",
	})
	require.EqualError(t, err, "Unknown proxy adapter type")

	_, err = ParseProxy(map[string]any{
		"type": "direct",
		"name": "DIRECT",
	})
	require.NoError(t, err)
}
