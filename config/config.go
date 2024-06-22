package config

import (
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"

	"github.com/lumavpn/luma/log"
	"gopkg.in/yaml.v3"
)

type Config struct {
	// General configuration
	LogLevel log.LogLevel `yaml:"loglevel"`
	IPv6     bool         `json:"ipv6" yaml:"ipv6"`
	// Use this interface when configuring the tunnel
	Interface string `yaml:"interface-name"`

	LanAllowedIPs    []netip.Prefix `yaml:"lan-allowed-ips"`
	LanDisAllowedIPs []netip.Prefix `yaml:"lan-disallowed-ips"`

	SocksPort int `yaml:"socks-port"`

	// Set firewall MARK (Linux only)
	Mark      int              `yaml:"mark"`
	Listeners []map[string]any `yaml:"listeners"`
	Proxies   []map[string]any `yaml:"proxies"`

	// Rules
	Rules    []string            `yaml:"rules"`
	SubRules map[string][]string `yaml:"sub-rules"`

	// Tunnel config
	Tun Tun `yaml:"tun"`
}

// New returns a new instance of Config with default values
func New() *Config {
	return &Config{
		LogLevel: log.DebugLevel,
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
