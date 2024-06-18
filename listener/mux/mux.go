package mux

import (
	"context"
	"net"
	"time"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/proxy/inbound"
	"github.com/lumavpn/luma/proxy/protos"
)

type ListenerConfig struct {
	Tunnel     adapter.TransportHandler
	Type       protos.Protocol
	Options    []inbound.Option
	UDPTimeout time.Duration
	MuxOption  MuxOption
}

type MuxOption struct {
	Padding bool          `yaml:"padding" json:"padding,omitempty"`
	Brutal  BrutalOptions `yaml:"brutal" json:"brutal,omitempty"`
}

type BrutalOptions struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Up      string `yaml:"up" json:"up,omitempty"`
	Down    string `yaml:"down" json:"down,omitempty"`
}

type ListenerHandler struct {
	ListenerConfig
}

func NewListenerHandler(lc ListenerConfig) (h *ListenerHandler, err error) {
	h = &ListenerHandler{ListenerConfig: lc}
	return
}

func (h *ListenerHandler) NewConnection(ctx context.Context, conn net.Conn, metadata M.Metadata) error {

}
