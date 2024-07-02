package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/stack"
	"gopkg.in/yaml.v3"
)

// Config is the Luma config manager
type Config struct {
	// General configuration
	General `yaml:",inline"`

	*Tun `yaml:"tun"`
}

// General configuration
type General struct {
	Inbound  `yaml:",inline"`
	LogLevel log.LogLevel `yaml:"log-level"`
	IPv6     bool         `json:"ipv6" yaml:"ipv6"`
}

// Inbound configuration
type Inbound struct {
	SocksPort   int    `yaml:"socks-port"`
	AllowLan    bool   `yaml:"allow-lan"`
	BindAddress string `yaml:"bind-address"`
}

// Tun configuration
type Tun struct {
	Enable    bool            `yaml:"enable" json:"enable"`
	Stack     stack.StackType `yaml:"stack" json:"stack"`
	DNSHijack []string        `yaml:"dns-hijack" json:"dns-hijack"`
	AutoRoute bool            `yaml:"auto-route" json:"auto-route"`
}

// New returns a new instance of Config with default values
func New() *Config {
	return new(Config)
}

// SetDefaultValues updates the given configuration to use default values
func (c *Config) SetDefaultValues() {
	if c.Tun == nil {
		c.Tun = new(Tun)
	}

}

// Init initializes a new Config from the given file and checks that it is valid
func Init(configFile string) (*Config, error) {
	if configFile == "" {
		return nil, errors.New("Missing config file")
	}
	if !filepath.IsAbs(configFile) {
		currentDir, _ := os.Getwd()
		configFile = filepath.Join(currentDir, configFile)
	}
	cfg, err := ParseConfig(configFile)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate checks if the given config is valid. It returns an error otherwise
func (c *Config) Validate() error {
	switch c.LogLevel.String() {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("unsupported loglevel:%s", c.LogLevel.String())
	}
	return nil
}

// ParseBytes unmarshals the given bytes into a Config
func ParseBytes(data []byte) (*Config, error) {
	cfg := New()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ParseConfig parses the config (if any) at the given path
func ParseConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return ParseBytes(data)
}
