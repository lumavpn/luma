package rule

import (
	"github.com/lumavpn/luma/metadata"
)

type NetworkType struct {
	*Base
	network metadata.Network
}

func NewNetworkType(network, adapter string) (*NetworkType, error) {
	ntType := NetworkType{
		Base: NewBase(RuleType_NETWORK, adapter),
	}

	ntType.adapter = adapter
	var err error
	ntType.network, err = metadata.EncodeNetwork(network)
	if err != nil {
		return nil, err
	}
	return &ntType, nil
}

func (n *NetworkType) Match(m *metadata.Metadata) (bool, string) {
	return n.network == m.Network, n.adapter
}

func (n *NetworkType) Payload() string {
	return n.network.String()
}
