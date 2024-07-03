package inbound

import (
	"fmt"

	"github.com/lumavpn/luma/adapter"
	LT "github.com/lumavpn/luma/listener/tunnel"
	"github.com/lumavpn/luma/log"
)

type TunnelOption struct {
	BaseOption
	Network []string `inbound:"network"`
	Target  string   `inbound:"target"`
}

func (o TunnelOption) Equal(config InboundConfig) bool {
	return optionToString(o) == optionToString(config)
}

type Tunnel struct {
	*Base
	config *TunnelOption
	ttl    *LT.Listener
	tul    *LT.PacketConn
}

func NewTunnel(options *TunnelOption) (*Tunnel, error) {
	base, err := NewBase(&options.BaseOption)
	if err != nil {
		return nil, err
	}
	return &Tunnel{
		Base:   base,
		config: options,
	}, nil
}

// Config implements constant.InboundListener
func (t *Tunnel) Config() InboundConfig {
	return t.config
}

// Close implements constant.InboundListener
func (t *Tunnel) Close() error {
	var err error
	if t.ttl != nil {
		if tcpErr := t.ttl.Close(); tcpErr != nil {
			err = tcpErr
		}
	}
	if t.tul != nil {
		if udpErr := t.tul.Close(); udpErr != nil {
			if err == nil {
				err = udpErr
			} else {
				return fmt.Errorf("close tcp err: %s, close udp err: %s", err.Error(), udpErr.Error())
			}
		}
	}

	return err
}

// Address implements constant.InboundListener
func (t *Tunnel) Address() string {
	if t.ttl != nil {
		return t.ttl.Address()
	}
	if t.tul != nil {
		return t.tul.Address()
	}
	return ""
}

// Listen implements constant.InboundListener
func (t *Tunnel) Listen(tunnel adapter.TransportHandler) error {
	var err error
	for _, network := range t.config.Network {
		switch network {
		case "tcp":
			if t.ttl, err = LT.New(t.RawAddress(), t.config.Target, t.config.SpecialProxy, tunnel, t.Additions()...); err != nil {
				return err
			}
		case "udp":
			if t.tul, err = LT.NewUDP(t.RawAddress(), t.config.Target, t.config.SpecialProxy, tunnel, t.Additions()...); err != nil {
				return err
			}
		default:
			log.Warnf("unknown network type: %s, passed", network)
			continue
		}
		log.Infof("Tunnel[%s](%s/%s)proxy listening at: %s", t.Name(), network, t.config.Target, t.Address())
	}
	return nil
}

var _ InboundListener = (*Tunnel)(nil)
