package proxydialer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProxyDialer_AddProxies(t *testing.T) {
	pd := New()
	err := pd.AddProxies(nil)
	require.NoError(t, err)
}
