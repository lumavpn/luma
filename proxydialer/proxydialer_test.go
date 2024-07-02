package proxydialer

import "testing"

func TestProxyDialer_AddProxies(t *testing.T) {
	pd := New()
	pd.AddProxies()
}
