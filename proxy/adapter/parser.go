package adapter

import (
	"fmt"

	"github.com/lumavpn/luma/common/structure"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/outbound"
	"github.com/lumavpn/luma/proxy/proto"
)

func ParseProxy(mapping map[string]any) (proxy.Proxy, error) {
	decoder := structure.NewDecoder(structure.Option{TagName: "proxy", WeaklyTypedInput: true, KeyReplacer: structure.DefaultKeyReplacer})
	proxyType, existType := mapping["type"].(string)
	if !existType {
		return nil, fmt.Errorf("missing type")
	}
	proxyProtocol, err := proto.EncodeProto(proxyType)
	if err != nil {
		return nil, err
	}
	var p proxy.ProxyAdapter
	switch proxyProtocol {
	case proto.Proto_DIRECT:
		return outbound.NewDirect(), nil
	case proto.Proto_HTTP:
		httpOption := &outbound.HttpOption{}
		err = decoder.Decode(mapping, httpOption)
		if err != nil {
			break
		}
		p, err = outbound.NewHTTP(*httpOption)
	case proto.Proto_SOCKS5:
		socksOption := &outbound.Socks5Option{}
		err = decoder.Decode(mapping, socksOption)
		if err != nil {
			break
		}
		p, err = outbound.NewSocks5(socksOption)
	default:
		return nil, fmt.Errorf("Unsupported protocol: %s", proxyProtocol)
	}
	if err != nil {
		return nil, err
	}
	return NewProxy(p), nil
}
