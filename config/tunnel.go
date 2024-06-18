package config

import (
	"net/netip"

	"github.com/lumavpn/luma/stack"
)

type Tun struct {
	Device              string          `yaml:"device" json:"device"`
	DNSHijack           []string        `yaml:"dns-hijack" json:"dns-hijack"`
	Enable              bool            `yaml:"enable" json:"enable"`
	AutoRoute           bool            `yaml:"auto-route" json:"auto-route"`
	AutoDetectInterface bool            `yaml:"auto-detect-interface"`
	Inet4Address        []netip.Prefix  `yaml:"inet4-address" json:"inet4-address,omitempty"`
	Inet6Address        []netip.Prefix  `yaml:"inet6-address" json:"inet6-address,omitempty"`
	MTU                 uint32          `yaml:"mtu" json:"mtu,omitempty"`
	Stack               stack.StackType `yaml:"stack" json:"stack"`
}
