package inbound

import (
	"errors"
	"strings"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/config"
	LC "github.com/lumavpn/luma/listener/config"
	"github.com/lumavpn/luma/listener/tun"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/stack"
)

type TunOption struct {
	BaseOption
	Device              string   `inbound:"device,omitempty"`
	Stack               string   `inbound:"stack,omitempty"`
	DNSHijack           []string `inbound:"dns-hijack,omitempty"`
	AutoRoute           bool     `inbound:"auto-route,omitempty"`
	AutoDetectInterface bool     `inbound:"auto-detect-interface,omitempty"`

	MTU                      uint32   `inbound:"mtu,omitempty"`
	GSO                      bool     `inbound:"gso,omitempty"`
	GSOMaxSize               uint32   `inbound:"gso-max-size,omitempty"`
	Inet4Address             []string `inbound:"inet4_address,omitempty"`
	Inet6Address             []string `inbound:"inet6_address,omitempty"`
	StrictRoute              bool     `inbound:"strict_route,omitempty"`
	Inet4RouteAddress        []string `inbound:"inet4_route_address,omitempty"`
	Inet6RouteAddress        []string `inbound:"inet6_route_address,omitempty"`
	Inet4RouteExcludeAddress []string `inbound:"inet4_route_exclude_address,omitempty"`
	Inet6RouteExcludeAddress []string `inbound:"inet6_route_exclude_address,omitempty"`
	IncludeInterface         []string `inbound:"include-interface,omitempty"`
	ExcludeInterface         []string `inbound:"exclude-interface" json:"exclude-interface,omitempty"`
	IncludeUID               []uint32 `inbound:"include_uid,omitempty"`
	IncludeUIDRange          []string `inbound:"include_uid_range,omitempty"`
	ExcludeUID               []uint32 `inbound:"exclude_uid,omitempty"`
	ExcludeUIDRange          []string `inbound:"exclude_uid_range,omitempty"`
	IncludeAndroidUser       []int    `inbound:"include_android_user,omitempty"`
	IncludePackage           []string `inbound:"include_package,omitempty"`
	ExcludePackage           []string `inbound:"exclude_package,omitempty"`
	EndpointIndependentNat   bool     `inbound:"endpoint_independent_nat,omitempty"`
	UDPTimeout               int64    `inbound:"udp_timeout,omitempty"`
	FileDescriptor           int      `inbound:"file-descriptor,omitempty"`
	TableIndex               int      `inbound:"table-index,omitempty"`
}

func (o TunOption) Equal(config InboundConfig) bool {
	return optionToString(o) == optionToString(config)
}

type Tun struct {
	*Base
	config *TunOption
	l      *tun.Listener
	tun    *config.Tun
}

func NewTun(options *TunOption) (*Tun, error) {
	base, err := NewBase(&options.BaseOption)
	if err != nil {
		return nil, err
	}
	stack, exist := stack.StackTypeMapping[strings.ToLower(options.Stack)]
	if !exist {
		return nil, errors.New("invalid tun stack")
	}
	inet4Address, err := LC.StringSliceToNetipPrefixSlice(options.Inet4Address)
	if err != nil {
		return nil, err
	}
	inet6Address, err := LC.StringSliceToNetipPrefixSlice(options.Inet6Address)
	if err != nil {
		return nil, err
	}
	inet4RouteAddress, err := LC.StringSliceToNetipPrefixSlice(options.Inet4RouteAddress)
	if err != nil {
		return nil, err
	}
	inet6RouteAddress, err := LC.StringSliceToNetipPrefixSlice(options.Inet6RouteAddress)
	if err != nil {
		return nil, err
	}
	inet4RouteExcludeAddress, err := LC.StringSliceToNetipPrefixSlice(options.Inet4RouteExcludeAddress)
	if err != nil {
		return nil, err
	}
	inet6RouteExcludeAddress, err := LC.StringSliceToNetipPrefixSlice(options.Inet6RouteExcludeAddress)
	if err != nil {
		return nil, err
	}
	return &Tun{
		Base:   base,
		config: options,
		tun: &config.Tun{
			Enable:                   true,
			Device:                   options.Device,
			Stack:                    stack,
			DNSHijack:                options.DNSHijack,
			AutoRoute:                options.AutoRoute,
			AutoDetectInterface:      options.AutoDetectInterface,
			MTU:                      options.MTU,
			GSO:                      options.GSO,
			GSOMaxSize:               options.GSOMaxSize,
			Inet4Address:             inet4Address,
			Inet6Address:             inet6Address,
			StrictRoute:              options.StrictRoute,
			Inet4RouteAddress:        inet4RouteAddress,
			Inet6RouteAddress:        inet6RouteAddress,
			Inet4RouteExcludeAddress: inet4RouteExcludeAddress,
			Inet6RouteExcludeAddress: inet6RouteExcludeAddress,
			IncludeInterface:         options.IncludeInterface,
			ExcludeInterface:         options.ExcludeInterface,
			IncludeUID:               options.IncludeUID,
			IncludeUIDRange:          options.IncludeUIDRange,
			ExcludeUID:               options.ExcludeUID,
			ExcludeUIDRange:          options.ExcludeUIDRange,
			IncludeAndroidUser:       options.IncludeAndroidUser,
			IncludePackage:           options.IncludePackage,
			ExcludePackage:           options.ExcludePackage,
			EndpointIndependentNat:   options.EndpointIndependentNat,
			UDPTimeout:               options.UDPTimeout,
			FileDescriptor:           options.FileDescriptor,
			TableIndex:               options.TableIndex,
		},
	}, nil
}

// Config implements constant.InboundListener
func (t *Tun) Config() InboundConfig {
	return t.config
}

// Address implements constant.InboundListener
func (t *Tun) Address() string {
	return t.l.Address()
}

// Listen implements constant.InboundListener
func (t *Tun) Listen(tunnel adapter.TransportHandler) error {
	var err error
	t.l, err = tun.New(t.tun, tunnel, t.Additions()...)
	if err != nil {
		return err
	}
	log.Infof("Tun[%s] proxy listening at: %s", t.Name(), t.Address())
	return nil
}

// Close implements constant.InboundListener
func (t *Tun) Close() error {
	return t.l.Close()
}

var _ InboundListener = (*Tun)(nil)
