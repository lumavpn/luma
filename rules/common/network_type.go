package common

import (
	"fmt"
	"strings"

	"github.com/lumavpn/luma/log"
	M "github.com/lumavpn/luma/metadata"
	R "github.com/lumavpn/luma/rule"
)

type NetworkType struct {
	*Base
	network M.Network
	adapter string
}

func NewNetworkType(network, adapter string) (*NetworkType, error) {
	ntType := NetworkType{
		Base: &Base{},
	}

	ntType.adapter = adapter
	switch strings.ToUpper(network) {
	case "TCP":
		ntType.network = M.TCP
	case "UDP":
		ntType.network = M.UDP
	default:
		return nil, fmt.Errorf("unsupported network type, only TCP/UDP")
	}

	return &ntType, nil
}

func (n *NetworkType) RuleType() R.RuleType {
	return R.Network
}

func (n *NetworkType) Match(metadata *M.Metadata) (bool, string) {
	log.Debug("Network type rule Match")
	return n.network == metadata.Network, n.adapter
}

func (n *NetworkType) Adapter() string {
	return n.adapter
}

func (n *NetworkType) Payload() string {
	return n.network.String()
}
