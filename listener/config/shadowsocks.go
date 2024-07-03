package config

import (
	"github.com/lumavpn/luma/listener/mux"

	"encoding/json"
)

type ShadowsocksServer struct {
	Enable    bool
	Listen    string
	Password  string
	Cipher    string
	Udp       bool
	MuxOption mux.MuxOption `yaml:"mux-option" json:"mux-option,omitempty"`
}

func (t ShadowsocksServer) String() string {
	b, _ := json.Marshal(t)
	return string(b)
}
