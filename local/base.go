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
	Addr         string `inbound:"addr"`
	NameStr      string `inbound:"name"`
	Listen       string `inbound:"listen,omitempty"`
	Port         int    `inbound:"port,omitempty"`
	SpecialRules string `inbound:"rule,omitempty"`
	SpecialProxy string `inbound:"proxy,omitempty"`
}

func NewBase(opts *BaseOption) (*BaseServer, error) {
	var listenAddr netip.Addr
	var err error
	if opts.Addr != "" {
		host, portStr, err := net.SplitHostPort(opts.Addr)
		if err != nil {
			return nil, err
		}
		port, _ := strconv.Atoi(portStr)
		opts.Port = port
		listenAddr, err = netip.ParseAddr(host)
	} else if opts.Listen == "" {
		opts.Listen = "0.0.0.0"
	}
	if opts.Listen != "" {
		listenAddr, err = netip.ParseAddr(opts.Listen)
	}
	if err != nil {
		return nil, err
	}
	return &BaseServer{
		name:         opts.Name(),
		listenAddr:   listenAddr,
		specialRules: opts.SpecialRules,
		port:         opts.Port,
		config:       opts,
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
