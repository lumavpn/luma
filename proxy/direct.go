package proxy

import "github.com/lumavpn/luma/proxy/proto"

type Direct struct {
	*Base
}

// NewDirect creates a new direct dialer that bypasses proxying traffic
func NewDirect() *Direct {
	return &Direct{
		Base: &Base{
			proto: proto.Proto_Direct,
		},
	}
}
