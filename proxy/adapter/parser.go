package adapter

import (
	"fmt"

	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/outbound"
	"github.com/lumavpn/luma/proxy/protos"
	structure "github.com/mitchellh/mapstructure"
)

func decodeOptions[T any](mapping map[string]any) (*T, error) {
	result := new(T)
	decoder, err := structure.NewDecoder(&structure.DecoderConfig{
		Result: result,
	})
	if err != nil {
		return nil, err
	}

	err = decoder.Decode(mapping)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func ParseProxy(mapping map[string]any) (proxy.Proxy, error) {
	at, existType := mapping["type"].(string)
	if !existType {
		return nil, fmt.Errorf("missing type")
	}
	adapterType, err := protos.EncodeAdapterType(at)
	if err != nil {
		return nil, err
	}
	var proxy proxy.ProxyAdapter
	switch adapterType {
	case protos.AdapterType_Direct:
		directOption, err := decodeOptions[outbound.BaseOptions](mapping)
		if err != nil {
			return nil, err
		}
		proxy = outbound.NewDirectWithOptions(*directOption)
	case protos.AdapterType_Http:
		httpOption, err := decodeOptions[outbound.HttpOptions](mapping)
		if err != nil {
			return nil, err
		}
		proxy, err = outbound.NewHTTP(*httpOption)
	case protos.AdapterType_Socks5:
		socksOption, err := decodeOptions[outbound.Socks5Options](mapping)
		if err != nil {
			return nil, err
		}
		proxy, err = outbound.NewSocks5(*socksOption)
	}
	if err != nil {
		return nil, err
	}
	return NewProxy(proxy), nil
}
