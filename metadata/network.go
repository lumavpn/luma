package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	TCP Network = iota
	UDP
)

// NetworkMapping is a mapping for the Network enum
var NetworkMapping = map[string]Network{
	TCP.String(): TCP,
	UDP.String(): UDP,
}

type Network uint8

func EncodeNetwork(n string) (Network, error) {
	if network, ok := NetworkMapping[strings.ToLower(n)]; ok {
		return network, nil
	}
	return Network(0), fmt.Errorf("Unknown network: %v", n)
}

func (n Network) String() string {
	switch n {
	case TCP:
		return "tcp"
	case UDP:
		return "udp"
	default:
		return fmt.Sprintf("network(%d)", n)
	}
}

// UnmarshalJSON deserialize Network with json
func (l *Network) UnmarshalJSON(data []byte) error {
	var lvl string
	if err := json.Unmarshal(data, &lvl); err != nil {
		return err
	}

	level, exist := NetworkMapping[lvl]
	if !exist {
		return errors.New("invalid network")
	}
	*l = level
	return nil
}

// MarshalJSON serialize Network with json
func (l Network) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}

// UnmarshalYAML unserialize Network with yaml
func (e *Network) UnmarshalYAML(unmarshal func(any) error) error {
	var tp string
	if err := unmarshal(&tp); err != nil {
		return err
	}
	mode, exist := NetworkMapping[strings.ToLower(tp)]
	if !exist {
		return errors.New("invalid network")
	}
	*e = mode
	return nil
}

// MarshalJSON serialize Network with yaml
func (l Network) MarshalYAML() (any, error) {
	return l.String(), nil
}
