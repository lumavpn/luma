package adapter

import (
	"fmt"

	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/outbound"
	"github.com/lumavpn/luma/proxy/protos"
	structure "github.com/mitchellh/mapstructure"
)

func ParseProxy(mapping map[string]any) (proxy.Proxy, error) {
	directOption := &outbound.BasicOptions{}
	decoder, err := structure.NewDecoder(&structure.DecoderConfig{
		Result: directOption,
	})
	if err != nil {
		return nil, err
	}
	at, existType := mapping["type"].(string)
	if !existType {
		return nil, fmt.Errorf("missing type")
	}
	adapterType, err := protos.EncodeAdapterType(at)
	if err != nil {
		return nil, err
	}

	var proxy proxy.Proxy
	switch adapterType {
	case protos.AdapterType_Direct:
		err = decoder.Decode(mapping)
		if err != nil {
			break
		}
		proxy = outbound.NewDirectWithOptions(*directOption)
	}
	return proxy, nil
}
