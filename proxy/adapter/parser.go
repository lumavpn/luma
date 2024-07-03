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

	at, err := proto.EncodeProto(proxyType)
	if err != nil {
		return nil, err
	}

	var proxy proxy.ProxyAdapter

	switch at {
	case proto.Proto_Direct:
		directOption := &outbound.DirectOpts{}
		err = decoder.Decode(mapping, directOption)
		if err != nil {
			break
		}
		proxy = outbound.NewDirectWithOptions(*directOption)
	case proto.Proto_Socks5:
		socksOption := &outbound.Socks5Option{}
		err = decoder.Decode(mapping, socksOption)
		if err != nil {
			break
		}
		proxy, err = outbound.NewSocks5(socksOption)
	case proto.Proto_HTTP:
		httpOption := &outbound.HttpOption{}
		err = decoder.Decode(mapping, httpOption)
		if err != nil {
			break
		}
		proxy, err = outbound.NewHttp(*httpOption)
	case proto.Proto_Reject:
		rejectOption := &outbound.RejectOption{}
		err = decoder.Decode(mapping, rejectOption)
		if err != nil {
			break
		}
		proxy = outbound.NewRejectWithOption(*rejectOption)
	default:
		return nil, fmt.Errorf("unsupport proxy type: %s", proxyType)
	}

	if err != nil {
		return nil, err
	}
	return NewProxy(proxy), nil
}
