package adapter

import (
	"fmt"

	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/proto"
)

func ParseProxy(mapping map[string]any) (proxy.Proxy, error) {
	at, existType := mapping["type"].(string)
	if !existType {
		return nil, fmt.Errorf("missing type")
	}
	proxyProtocol, err := proto.EncodeProto(at)
	if err != nil {
		return nil, err
	}
	switch proxyProtocol {
	case proto.Proto_DIRECT:
		return proxy.NewDirect(), nil
	default:
		return nil, fmt.Errorf("Unsupported protocol: %s", proxyProtocol)
	}
}
