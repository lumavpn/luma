package local

import (
	"encoding/json"
	"net"
	"net/netip"
	"strconv"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/proxy/inbound"
)

type BaseServer struct {
	config       *BaseOption
	name         string
	specialRules string
	listenAddr   netip.Addr
	port         int
}

type BaseOption struct {
	NameStr      string `inbound:"name"`
	Listen       string `inbound:"listen,omitempty"`
	Port         int    `inbound:"port,omitempty"`
	SpecialRules string `inbound:"rule,omitempty"`
	SpecialProxy string `inbound:"proxy,omitempty"`
}

func NewBase(options *BaseOption) (*BaseServer, error) {
	if options.Listen == "" {
		options.Listen = "0.0.0.0"
	}
	addr, err := netip.ParseAddr(options.Listen)
	if err != nil {
		return nil, err
	}
	return &BaseServer{
		name:         options.Name(),
		listenAddr:   addr,
		specialRules: options.SpecialRules,
		port:         options.Port,
		config:       options,
	}, nil
}

// Config is the configuration of the local server
func (b *BaseServer) Config() LocalConfig {
	return b.config
}

func (b *BaseServer) Address() string {
	return b.RawAddress()
}

func (*BaseServer) Close() error {
	return nil
}

func (b *BaseServer) Name() string {
	return b.name
}

func (b *BaseServer) RawAddress() string {
	return net.JoinHostPort(b.listenAddr.String(), strconv.Itoa(int(b.port)))
}

func (*BaseServer) Listen(tunnel adapter.TransportHandler) error {
	return nil
}

func (b *BaseServer) Additions() []inbound.Option {
	o := b.config
	return []inbound.Option{
		inbound.WithInName(o.NameStr),
		inbound.WithSpecialRules(o.SpecialRules),
		inbound.WithSpecialProxy(o.SpecialProxy),
	}
}

func (o BaseOption) Name() string {
	return o.NameStr
}

func (o BaseOption) Options() []inbound.Option {
	return []inbound.Option{
		inbound.WithInName(o.NameStr),
		inbound.WithSpecialRules(o.SpecialRules),
		inbound.WithSpecialProxy(o.SpecialProxy),
	}
}

func (o BaseOption) Equal(config LocalConfig) bool {
	return optionToString(o) == optionToString(config)
}

func optionToString(option any) string {
	str, _ := json.Marshal(option)
	return string(str)
}
