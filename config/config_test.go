package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Clone(t *testing.T) {
	c := &Config{
		Inbound: Inbound{
			SocksPort: 8787,
		},
		LogLevel: "debug",
		Proxy:    "socksproxy",
	}
	cloned := c.Clone()
	assert.Equal(t, "socksproxy", cloned.Proxy)

	c.Proxy = "httpproxy"
	assert.Equal(t, "socksproxy", cloned.Proxy)
}
