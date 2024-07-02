package config

import (
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"

	C "github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/stack"
	"gopkg.in/yaml.v3"
)

// Config is the Luma config manager
type Config struct {
	// General configuration
	General `yaml:",inline"`

	Mode C.TunnelMode `yaml:"mode"`

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
	Enable                   bool            `yaml:"enable" json:"enable"`
	Device                   string          `yaml:"device" json:"device"`
	Interface                string          `yaml:"interface" json:"interface"`
	Stack                    stack.StackType `yaml:"stack" json:"stack"`
	DNSHijack                []string        `yaml:"dns-hijack" json:"dns-hijack"`
	AutoRoute                bool            `yaml:"auto-route" json:"auto-route"`
	AutoDetectInterface      bool            `yaml:"auto-detect-interface" json:"auto-detect-interface"`
	RedirectToTun            []string        `yaml:"-" json:"-"`
	DisableInterfaceMonitor  bool            `yaml:"disable-interface-monitor"`
	BuildAndroidRules        bool            `yaml:"build-android-rules"`
	MTU                      uint32          `yaml:"mtu" json:"mtu,omitempty"`
	GSO                      bool            `yaml:"gso" json:"gso,omitempty"`
	GSOMaxSize               uint32          `yaml:"gso-max-size" json:"gso-max-size,omitempty"`
	Inet4Address             []netip.Prefix  `yaml:"inet4-address" json:"inet4-address,omitempty"`
	Inet6Address             []netip.Prefix  `yaml:"inet6-address" json:"inet6-address,omitempty"`
	StrictRoute              bool            `yaml:"strict-route" json:"strict-route,omitempty"`
	Inet4RouteAddress        []netip.Prefix  `yaml:"inet4-route-address" json:"inet4-route-address,omitempty"`
	Inet6RouteAddress        []netip.Prefix  `yaml:"inet6-route-address" json:"inet6-route-address,omitempty"`
	Inet4RouteExcludeAddress []netip.Prefix  `yaml:"inet4-route-exclude-address" json:"inet4-route-exclude-address,omitempty"`
	Inet6RouteExcludeAddress []netip.Prefix  `yaml:"inet6-route-exclude-address" json:"inet6-route-exclude-address,omitempty"`
	IncludeInterface         []string        `yaml:"include-interface" json:"include-interface,omitempty"`
	ExcludeInterface         []string        `yaml:"exclude-interface" json:"exclude-interface,omitempty"`
	IncludeUID               []uint32        `yaml:"include-uid" json:"include-uid,omitempty"`
	IncludeUIDRange          []string        `yaml:"include-uid-range" json:"include-uid-range,omitempty"`
	ExcludeUID               []uint32        `yaml:"exclude-uid" json:"exclude-uid,omitempty"`
	ExcludeUIDRange          []string        `yaml:"exclude-uid-range" json:"exclude-uid-range,omitempty"`
	IncludeAndroidUser       []int           `yaml:"include-android-user" json:"include-android-user,omitempty"`
	IncludePackage           []string        `yaml:"include-package" json:"include-package,omitempty"`
	ExcludePackage           []string        `yaml:"exclude-package" json:"exclude-package,omitempty"`
	EndpointIndependentNat   bool            `yaml:"endpoint-independent-nat" json:"endpoint-independent-nat,omitempty"`
	UDPTimeout               int64           `yaml:"udp-timeout" json:"udp-timeout,omitempty"`
	FileDescriptor           int             `yaml:"file-descriptor" json:"file-descriptor"`
	TableIndex               int             `yaml:"table-index" json:"table-index"`
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
