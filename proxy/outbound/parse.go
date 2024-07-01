package proxy

import (
	"fmt"

	"github.com/lumavpn/luma/common/structure"
	"github.com/lumavpn/luma/proxy/adapter"
	"github.com/lumavpn/luma/proxy/proto"
)

func ParseProxy(mapping map[string]any) (Proxy, error) {
	decoder := structure.NewDecoder(structure.Option{TagName: "proxy", WeaklyTypedInput: true, KeyReplacer: structure.DefaultKeyReplacer})
	proxyType, existType := mapping["type"].(string)
	if !existType {
		return nil, fmt.Errorf("missing type")
	}
	proxyProtocol, err := proto.EncodeProto(proxyType)
	if err != nil {
		return nil, err
	}
	var p adapter.ProxyAdapter
	switch proxyProtocol {
	case proto.Proto_DIRECT:
		return NewDirect(), nil
	case proto.Proto_HTTP:
		httpOption := &HttpOption{}
		err = decoder.Decode(mapping, httpOption)
		if err != nil {
			break
		}
		p, err = NewHTTP(*httpOption)
	case proto.Proto_SOCKS5:
		socksOption := &Socks5Option{}
		err = decoder.Decode(mapping, socksOption)
		if err != nil {
			break
		}
		p, err = NewSocks5(socksOption)
	default:
		return nil, fmt.Errorf("Unsupported protocol: %s", proxyProtocol)
	}
	if err != nil {
		return nil, err
	}
	return adapter.NewProxy(p), nil
}
