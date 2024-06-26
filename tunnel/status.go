package tunnel

import (
	"encoding/json"
	"errors"
	"strings"
)

const (
	Suspend TunnelStatus = iota
	Inner
	Running
)

type TunnelStatus int

// StatusMapping is a mapping for the TunnelStatus enum
var StatusMapping = map[string]TunnelStatus{
	Suspend.String(): Suspend,
	Inner.String():   Inner,
	Running.String(): Running,
}

// UnmarshalJSON unserialize TunnelStatus
func (s *TunnelStatus) UnmarshalJSON(data []byte) error {
	var tp string
	json.Unmarshal(data, &tp)
	status, exist := StatusMapping[strings.ToLower(tp)]
	if !exist {
		return errors.New("invalid mode")
	}
	*s = status
	return nil
}

// UnmarshalYAML unserialize TunnelStatus with yaml
func (s *TunnelStatus) UnmarshalYAML(unmarshal func(any) error) error {
	var tp string
	unmarshal(&tp)
	status, exist := StatusMapping[strings.ToLower(tp)]
	if !exist {
		return errors.New("invalid status")
	}
	*s = status
	return nil
}

// MarshalJSON serialize TunnelStatus
func (s TunnelStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// MarshalYAML serialize TunnelMode with yaml
func (s TunnelStatus) MarshalYAML() (any, error) {
	return s.String(), nil
}

func (s TunnelStatus) String() string {
	switch s {
	case Inner:
		return "inner"
	case Suspend:
		return "suspend"
	case Running:
		return "unknown"
	default:
		return "Unknown"
	}
}
