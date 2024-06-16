package listener

import (
	"fmt"

	"github.com/lumavpn/luma/listener/inbound"
	structure "github.com/mitchellh/mapstructure"
)

func ParseListener(mapping map[string]any) (inbound.InboundListener, error) {
	socksOption := &inbound.SocksOption{UDP: true}
	decoder, err := structure.NewDecoder(&structure.DecoderConfig{
		Result: socksOption,
	})
	if err != nil {
		return nil, err
	}

	proxyType, existType := mapping["type"].(string)
	if !existType {
		return nil, fmt.Errorf("missing type")
	}

	var listener inbound.InboundListener

	switch proxyType {
	case "socks":
		err = decoder.Decode(mapping)
		if err != nil {
			return nil, err
		}
		listener, err = inbound.NewSocks(socksOption)
	default:
		return nil, fmt.Errorf("unknown proxy type: %s", proxyType)
	}
	return listener, err
}
