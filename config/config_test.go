package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParseConfig(t *testing.T) {
	_, err := ParseConfig("")
	require.True(t, os.IsNotExist(err))
}

func TestParseBytes(t *testing.T) {
	cfg, err := ParseBytes([]byte(`loglevel: invalid`))
	require.EqualError(t, err, "invalid log level")
	cfg, err = ParseBytes([]byte(`loglevel: debug`))
	require.NoError(t, err)
	b, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	fmt.Println(string(b))
}
