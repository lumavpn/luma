package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"

	C "github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/geodata"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/stack"
	"github.com/lumavpn/luma/util"
	"gopkg.in/yaml.v3"
)

type Inbound struct {
	LanAllowedIPs    []netip.Prefix `json:"lan-allowed-ips" yaml:"lan-allowed-ips"`
	LanDisAllowedIPs []netip.Prefix `json:"lan-disallowed-ips" yaml:"lan-disallowed-ips"`
	AllowLan         bool           `json:"allow-lan" yaml:"allow-lan"`
	BindAddress      string         `json:"bind-address" yaml:"bind-address"`
	SkipAuthPrefixes []netip.Prefix `json:"skip-auth-prefixes"`
	Port             int            `yaml:"port" json:"port"`
	SocksPort        int            `yaml:"socks-port" json:"socks-port"`
	RedirPort        int            `yaml:"redir-port" json:"redir-port"`
	TProxyPort       int            `yaml:"tproxy-port" json:"tproxy-port"`
	MixedPort        int            `yaml:"mixed-port" json:"mixed-port"`
	InboundTfo       bool           `json:"inbound-tfo"`
	InboundMPTCP     bool           `json:"inbound-mptcp"`
}

type GeoConfig struct {
	GeoAutoUpdate     bool   `yaml:"geo-auto-update" json:"geo-auto-update"`
	GeoUpdateInterval int    `yaml:"geo-update-interval" json:"geo-update-interval"`
	GeodataMode       bool   `yaml:"geodata-mode" json:"geodata-mode"`
	GeodataLoader     string `yaml:"geodata-loader" json:"geodata-loader"`
	GeositeMatcher    string `yaml:"geosite-matcher" json:"geosite-matcher"`
}

type GeoXUrl struct {
	GeoIp   string `yaml:"geoip" json:"geoip"`
	Mmdb    string `yaml:"mmdb" json:"mmdb"`
	ASN     string `yaml:"asn" json:"asn"`
	GeoSite string `yaml:"geosite" json:"geosite"`
}

type General struct {
	Inbound       `yaml:",inline"`
	GeoConfig     `yaml:",inline"`
	GeoXUrl       *GeoXUrl `yaml:"geox-url"`
	LogLevel      string   `yaml:"log-level"`
	GlobalUA      string   `yaml:"global-ua"`
	IPv6          bool     `json:"ipv6" yaml:"ipv6"`
	TCPConcurrent bool     `yaml:"tcp-concurrent" json:"tcp-concurrent"`
}

// Config represents the options available for configuring an instance of Luma
type Config struct {
	General `yaml:",inline"`

	// Use this device [driver://]name
	Device string `json:"device,omitempty" yaml:"device"`

	Interface string       `yaml:"interface-name"`
	Mode      C.TunnelMode `yaml:"mode"`

	EnableSystemProxy bool `yaml:"enable-system-proxy"`

	// Set firewall MARK (Linux only)
	Mark int

	Profile *Profile `yaml:"profile,omitempty"`

	EnableTun2socks bool   `yaml:"enable-tun2socks"`
	Proxy           string `yaml:"proxy"`

	RawProxies []map[string]any `yaml:"proxies"`
	//Proxies    map[string]P.Proxy `yaml:"-"`

	//Listeners    map[string]IN.InboundListener `yaml:"-"`
	RawListeners []map[string]any `yaml:"listeners"`

	ProxyGroup []map[string]any `yaml:"proxy-groups"`

	Rules    []string            `yaml:"rules"`
	SubRules map[string][]string `yaml:"sub-rules"`

	DNS    *DNS   `yaml:"-"`
	RawDNS RawDNS `yaml:"dns" json:"dns"`

	Hosts map[string]any `yaml:"hosts" json:"hosts"`

	RawSniffer RawSniffer `yaml:"sniffer" json:"sniffer"`
	Sniffer    *Sniffer   `yaml:"-" json:"-"`

	ProxyProvider map[string]map[string]any `yaml:"proxy-providers"`

	RuleProviders map[string]map[string]any `yaml:"rule-providers"`

	MTU int `yaml:"mtu"`

	EBpf   EBpf   `yaml:"ebpf"`
	RawTun RawTun `yaml:"tun"`
	Tun    *Tun   `yaml:"-"`
}

// Profile config
type Profile struct {
	StoreSelected bool `yaml:"store-selected"`
	StoreFakeIP   bool `yaml:"store-fake-ip"`
}

// New returns a new instance of Config with default values
func New() *Config {
	return &Config{
		General: General{
			LogLevel:      "info",
			IPv6:          true,
			TCPConcurrent: false,
			Inbound: Inbound{
				AllowLan:      false,
				BindAddress:   "*",
				LanAllowedIPs: []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0"), netip.MustParsePrefix("::/0")},
			},
			GeoXUrl: &GeoXUrl{
				Mmdb:    "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.metadb",
				ASN:     "https://github.com/xishang0128/geoip/releases/download/latest/GeoLite2-ASN.mmdb",
				GeoIp:   "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.dat",
				GeoSite: "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geosite.dat",
			},
		},
		Mode:       C.Rule,
		Hosts:      map[string]any{},
		Rules:      []string{},
		RawProxies: []map[string]any{},
		Profile: &Profile{
			StoreSelected: true,
		},
		RawDNS: RawDNS{
			Enable:         false,
			IPv6:           false,
			UseHosts:       true,
			UseSystemHosts: true,
			IPv6Timeout:    100,
			EnhancedMode:   C.DNSMapping,
			FakeIPRange:    "198.18.0.1/16",
			FallbackFilter: RawFallbackFilter{
				GeoIP:     true,
				GeoIPCode: "CN",
				IPCIDR:    []string{},
				GeoSite:   []string{},
			},
			DefaultNameserver: []string{
				"114.114.114.114",
				"223.5.5.5",
				"8.8.8.8",
				"1.0.0.1",
			},
			NameServer: []string{
				"https://doh.pub/dns-query",
				"tls://223.5.5.5:853",
			},
			FakeIPFilter: []string{
				"dns.msftnsci.com",
				"www.msftnsci.com",
				"www.msftconnecttest.com",
			},
		},
		RawSniffer: RawSniffer{
			Enable:          false,
			Sniffing:        []string{},
			ForceDomain:     []string{},
			SkipDomain:      []string{},
			Ports:           []string{},
			ForceDnsMapping: true,
			ParsePureIp:     true,
			OverrideDest:    true,
		},
		RawTun: RawTun{
			Enable: false,
			Device: "",
			Stack:  stack.TunGVisor,
			//DNSHijack:           []string{"0.0.0.0:53"},
			AutoRoute:           true,
			AutoDetectInterface: true,
			Inet6Address:        []netip.Prefix{netip.MustParsePrefix("fdfe:dcba:9876::1/126")},
		},
	}
}

func Init(configFile string, cmdConfig *Config) *Config {
	var cfg *Config
	if configFile != "" {
		if !filepath.IsAbs(configFile) {
			currentDir, _ := os.Getwd()
			configFile = filepath.Join(currentDir, configFile)
		}

		exists, err := util.FileExists(configFile)
		if !exists || err != nil {
			log.Fatalf("No config file found at %s: %v", configFile, err)
		}

		cfg, err = ParseConfig(configFile)
		if err != nil && !os.IsNotExist(err) {
			log.Fatal(err)
		} else if cmdConfig != nil {
			OverrideConfig(cfg, cmdConfig)
		}
	} else if cmdConfig != nil {
		cfg = cmdConfig
	}

	if cfg == nil {
		log.Fatal("Config missing")
	}

	if err := cfg.Validate(); err != nil {
		log.Errorf("config is invalid: %v", err)
		os.Exit(1)
	}
	return cfg
}

func (c *Config) Clone() *Config {
	b, _ := yaml.Marshal(c)
	cc := new(Config)
	_ = yaml.Unmarshal(b, cc)
	return cc
}

func unmarshalConfig(cfg *Config, data []byte) (*Config, error) {
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal config: %v", err)
	}

	geodata.SetGeodataMode(cfg.GeodataMode)
	geodata.SetGeoAutoUpdate(cfg.GeoAutoUpdate)
	geodata.SetGeoUpdateInterval(cfg.GeoUpdateInterval)
	geodata.SetLoader(cfg.GeodataLoader)
	geodata.SetSiteMatcher(cfg.GeositeMatcher)
	C.GeoAutoUpdate = cfg.GeoAutoUpdate
	C.GeoUpdateInterval = cfg.GeoUpdateInterval
	if cfg.GeoXUrl != nil {
		C.GeoIpUrl = cfg.GeoXUrl.GeoIp
		C.GeoSiteUrl = cfg.GeoXUrl.GeoSite
		C.MmdbUrl = cfg.GeoXUrl.Mmdb
		C.ASNUrl = cfg.GeoXUrl.ASN
	}
	C.GeodataMode = cfg.GeodataMode
	C.UA = cfg.GlobalUA

	if len(cfg.RawTun.Inet6Address) == 0 {
		cfg.RawTun.Inet6Address = []netip.Prefix{netip.MustParsePrefix("fdfe:dcba:9876::1/126")}
	}
	//b, _ := json.Marshal(cfg.DNS)
	//log.Debugf("Dns config is %s", string(b))
	var err error
	cfg.Sniffer, err = parseSniffer(cfg.RawSniffer)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

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

	cfg := New()
	return unmarshalConfig(cfg, data)
}

func ParseWithBytes(b []byte) (*Config, error) {
	cfg := New()
	return unmarshalConfig(cfg, b)
}

func OverrideConfig[T any](dst, src *T) {
	newVal := reflect.ValueOf(src).Elem()
	oldVal := reflect.ValueOf(dst).Elem()

	for i := 0; i < newVal.NumField(); i++ {
		srcField := newVal.Field(i)
		dstField := oldVal.Field(i)

		switch srcField.Kind() {
		case reflect.String:
			s := srcField.String()
			if s != "" {
				dstField.SetString(s)
			}
		case reflect.Int:
			i := srcField.Int()
			if i != 0 {
				dstField.SetInt(i)
			}
		case reflect.Bool:
			b := srcField.Bool()
			if b {
				dstField.SetBool(b)
			}
		}
	}
}

func prettyprint(b []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "  ")
	return out.Bytes(), err
}

func (c *Config) Validate() error {
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("unsupported log-level:%s, supported log-levels:[debug, info, warn, error]", c.LogLevel)
	}
	b, _ := json.Marshal(c)
	b, _ = prettyprint(b)
	log.Debugf("Config is %s", string(b))
	return nil
}
