package metadata

import (
	"errors"
	"strings"
)

const (
	TCP Network = iota
	UDP
	ALLNet
	InvalidNet = 0xff
)

var (
	errNetworkNotFound = errors.New("Network not found")
)

type Network uint8

// NetworkMapping is a mapping for Mode enum
var NetworkMapping = map[string]Network{
	TCP.String():    TCP,
	UDP.String():    UDP,
	ALLNet.String(): ALLNet,
}

func (n Network) String() string {
	switch n {
	case TCP:
		return "tcp"
	case UDP:
		return "udp"
	case ALLNet:
		return "all"
	default:
		return "invalid"
	}
}

func (n Network) MarshalText() ([]byte, error) {
	return []byte(n.String()), nil
}

func ParseNetwork(n string) (Network, error) {
	if network, ok := NetworkMapping[strings.ToLower(n)]; ok {
		return network, nil
	}
	return InvalidNet, errNetworkNotFound
}
