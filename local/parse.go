package local

import (
	"fmt"

	"github.com/lumavpn/luma/common/structure"
)

func ParseLocal(mapping map[string]any) (LocalServer, error) {
	decoder := structure.NewDecoder(structure.Option{TagName: "inbound", WeaklyTypedInput: true, KeyReplacer: structure.DefaultKeyReplacer})
	proxyType, existType := mapping["type"].(string)
	if !existType {
		return nil, fmt.Errorf("missing type")
	}

	var (
		server LocalServer
		err    error
	)
	switch proxyType {
	case "socks":
		socksOption := &SocksOption{UDP: true}
		err = decoder.Decode(mapping, socksOption)
		if err != nil {
			return nil, err
		}
		server, err = NewSocks(socksOption)
	default:
		return nil, fmt.Errorf("unsupport proxy type: %s", proxyType)
	}
	return server, err
}
