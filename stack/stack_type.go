package stack

import (
	"errors"
	"strings"
)

type StackType uint8

const (
	TunGVisor StackType = iota
	TunLWIP
	TunMixed
	TunSystem
)

var (
	// StackTypeMapping is a mapping for the StackType enum
	StackTypeMapping = map[string]StackType{
		TunGVisor.String(): TunGVisor,
		TunLWIP.String():   TunLWIP,
		TunMixed.String():  TunMixed,
		TunSystem.String(): TunSystem,
	}
)

// UnmarshalYAML unserialize StackType with yaml
func (e *StackType) UnmarshalYAML(unmarshal func(any) error) error {
	var tp string
	if err := unmarshal(&tp); err != nil {
		return err
	}
	mode, exist := StackTypeMapping[strings.ToLower(tp)]
	if !exist {
		return errors.New("invalid tun stack")
	}
	*e = mode
	return nil
}

// MarshalYAML serialize StackType with yaml
func (e StackType) MarshalYAML() (any, error) {
	return e.String(), nil
}

func (st StackType) String() string {
	switch st {
	case TunGVisor:
		return "gvisor"
	case TunLWIP:
		return "lwip"
	case TunMixed:
		return "mixed"
	case TunSystem:
		return "system"
	default:
		return ""
	}
}
