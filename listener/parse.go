package listener

import (
	"fmt"

	"github.com/lumavpn/luma/common/structure"
	IN "github.com/lumavpn/luma/listener/inbound"
	"github.com/lumavpn/luma/stack"
)

func ParseListener(mapping map[string]any) (IN.InboundListener, error) {
	decoder := structure.NewDecoder(structure.Option{TagName: "inbound", WeaklyTypedInput: true, KeyReplacer: structure.DefaultKeyReplacer})
	proxyType, existType := mapping["type"].(string)
	if !existType {
		return nil, fmt.Errorf("missing type")
	}

	var (
		listener IN.InboundListener
		err      error
	)
	switch proxyType {
	case "socks":
		socksOption := &IN.SocksOption{UDP: true}
		err = decoder.Decode(mapping, socksOption)
		if err != nil {
			return nil, err
		}
		listener, err = IN.NewSocks(socksOption)
	case "tunnel":
		tunnelOption := &IN.TunnelOption{}
		err = decoder.Decode(mapping, tunnelOption)
		if err != nil {
			return nil, err
		}
		listener, err = IN.NewTunnel(tunnelOption)
	case "tun":
		tunOption := &IN.TunOption{
			Stack:     stack.TunGVisor.String(),
			DNSHijack: []string{"0.0.0.0:53"}, // default hijack all dns query
		}
		err = decoder.Decode(mapping, tunOption)
		if err != nil {
			return nil, err
		}
		listener, err = IN.NewTun(tunOption)
	default:
		return nil, fmt.Errorf("unsupport proxy type: %s", proxyType)
	}
	return listener, err
}
