package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/log"
	"gopkg.in/yaml.v3"
)

type Config struct {
	// General configuration
	LogLevel log.LogLevel      `yaml:"loglevel"`
	BindAll  bool              `yaml:"bind-all" json:"bind_all"`
	Mode     common.TunnelMode `yaml:"mode" json:"mode"`
	// Set firewall MARK (Linux only)
	Mark int

	Hosts map[string]any `yaml:"hosts" json:"hosts"`

	// Proxies
	SocksPort int `yaml:"socks-port" json:"socks_port"`

	IPv6 bool `yaml:"ipv6"`

	Device    string `yaml:"device"`
	Interface string `yaml:"interface"`
	MTU       uint32 `yaml:"mtu"`

	Locals  []map[string]any `yaml:"locals"`
	Proxies []map[string]any `yaml:"proxies"`

	RawDNS RawDNS `yaml:"dns" json:"dns"`
	DNS    *DNS   `yaml:"-" json:"-"`
	Tun    *Tun   `yaml:"tun" json:"tun"`
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
