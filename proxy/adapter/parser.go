package adapter

import (
	"fmt"

	"github.com/lumavpn/luma/common/structure"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/proto"
)

func ParseProxy(mapping map[string]any) (proxy.Proxy, error) {
	decoder := structure.NewDecoder(structure.Option{TagName: "proxy", WeaklyTypedInput: true,
		KeyReplacer: structure.DefaultKeyReplacer})
	proxyType, existType := mapping["type"].(string)
	if !existType {
		return nil, fmt.Errorf("missing type")
	}
	proxyProtocol, err := proto.EncodeProto(proxyType)
	if err != nil {
		return nil, err
	}
	var p proxy.Proxy
	switch proxyProtocol {
	case proto.Proto_DIRECT:
		return proxy.NewDirect(), nil
	case proto.Proto_SOCKS5:
		socks5Option := &proxy.Socks5Option{}
		err = decoder.Decode(mapping, socks5Option)
		if err != nil {
			break
		}
		p, err = proxy.NewSocks5(socks5Option)
	default:
		return nil, fmt.Errorf("Unsupported protocol: %s", proxyProtocol)
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}
