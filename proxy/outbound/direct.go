package outbound

import (
	"context"
	"net"

	"github.com/lumavpn/luma/dialer"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/protos"
)

type Direct struct {
	*Base
}

// NewDirect returns a new instance of a direct outbound proxy
func NewDirect() *Direct {
	at := protos.AdapterType_Direct
	return &Direct{
		Base: &Base{
			name: at.String(),
			at:   at,
			udp:  true,
		},
	}
}

// NewDirectWithOptions returns a new instance of Direct configured with the given options
func NewDirectWithOptions(opts BasicOptions) *Direct {
	return &Direct{
		Base: &Base{
			interfaceName: opts.Interface,
			routingMark:   opts.RoutingMark,
			name:          opts.Name,
			at:            protos.AdapterType_Direct,
			udp:           true,
		},
	}
}

// DialContext connects to the address on the network using the provided Metadata
func (d *Direct) DialContext(ctx context.Context, metadata *M.Metadata) (net.Conn, error) {
	c, err := dialer.DialContext(ctx, "tcp", metadata.DestinationAddress())
	if err != nil {
		return nil, err
	}
	setKeepAlive(c)
	return c, nil
}
