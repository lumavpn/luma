package config

import (
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"

	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/stack"
	"gopkg.in/yaml.v3"
)

type Config struct {
	// General configuration
	LogLevel log.LogLevel `yaml:"loglevel"`

	Device    string `yaml:"device"`
	Interface string `yaml:"interface"`
	MTU       uint32 `yaml:"mtu"`

	Locals  []map[string]any `yaml:"locals"`
	Proxies []map[string]any `yaml:"proxies"`

	Tun TunConfig `yaml:"tun" json:"tun"`
}

type TunConfig struct {
	AutoRoute bool `yaml:"auto-route" json:"auto_route"`

	Enable    bool            `yaml:"enable" json:"enable"`
	Device    string          `yaml:"device" json:"device"`
	DNSHijack []string        `yaml:"dns-hijack" json:"dns-hijack"`
	Interface string          `yaml:"interface" json:"interface"`
	Stack     stack.StackType `yaml:"stack" json:"stack"`

	Inet4Address             []netip.Prefix `yaml:"inet4-address" json:"inet4-address,omitempty"`
	Inet6Address             []netip.Prefix `yaml:"inet6-address" json:"inet6-address,omitempty"`
	Inet4RouteAddress        []netip.Prefix `yaml:"inet4-route-address" json:"inet4_route_address,omitempty"`
	Inet6RouteAddress        []netip.Prefix `yaml:"inet6-route-address" json:"inet6_route_address,omitempty"`
	Inet4RouteExcludeAddress []netip.Prefix `yaml:"inet4-route-exclude-address" json:"inet4_route_exclude_address,omitempty"`
	Inet6RouteExcludeAddress []netip.Prefix `yaml:"inet6-route-exclude-address" json:"inet6_route_exclude_address,omitempty"`
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
