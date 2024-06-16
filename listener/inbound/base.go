package inbound

import (
	"encoding/json"
	"net"
	"net/netip"
	"strconv"

	"github.com/lumavpn/luma/adapter"
)

type Base struct {
	config       *BaseOption
	name         string
	specialRules string
	listenAddr   netip.Addr
	port         int
}

// NewBase creates a new Base inbound Listener
func NewBase(options *BaseOption) (*Base, error) {
	if options.Listen == "" {
		options.Listen = "0.0.0.0"
	}
	addr, err := netip.ParseAddr(options.Listen)
	if err != nil {
		return nil, err
	}
	return &Base{
		name:         options.Name(),
		listenAddr:   addr,
		specialRules: options.SpecialRules,
		port:         options.Port,
		config:       options,
	}, nil
}

// Config returns the config the InboundListener is configured with
func (b *Base) Config() InboundConfig {
	return b.config
}

// Address returns the address of the InboundListener
func (b *Base) Address() string {
	return b.RawAddress()
}

// Close the inbound listener
func (*Base) Close() error {
	return nil
}

// RawAddress returns the name of the InboundListener
func (b *Base) Name() string {
	return b.name
}

// RawAddress returns the raw address of the InboundListener
func (b *Base) RawAddress() string {
	return net.JoinHostPort(b.listenAddr.String(), strconv.Itoa(int(b.port)))
}

// Listen implements constant.InboundListener
func (*Base) Listen(tunnel adapter.TransportHandler) error {
	return nil
}

var _ InboundListener = (*Base)(nil)

type BaseOption struct {
	NameStr      string `inbound:"name"`
	Listen       string `inbound:"listen,omitempty"`
	Port         int    `inbound:"port,omitempty"`
	SpecialRules string `inbound:"rule,omitempty"`
	SpecialProxy string `inbound:"proxy,omitempty"`
}

func (o BaseOption) Name() string {
	return o.NameStr
}

func (o BaseOption) Equal(config InboundConfig) bool {
	return optionToString(o) == optionToString(config)
}

var _ InboundConfig = (*BaseOption)(nil)

func optionToString(option any) string {
	str, _ := json.Marshal(option)
	return string(str)
}
