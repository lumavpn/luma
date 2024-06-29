package provider

import (
	"errors"
	"strings"
)

type ProviderType int

const (
	Proxy ProviderType = iota
	Rule
)

var (
	// ProviderTypeMapping is a mapping for the ProviderType enum
	ProviderTypeMapping = map[string]ProviderType{
		Proxy.String(): Proxy,
		Rule.String():  Rule,
	}
)

// UnmarshalYAML unserialize ProviderType with yaml
func (e *ProviderType) UnmarshalYAML(unmarshal func(any) error) error {
	var tp string
	if err := unmarshal(&tp); err != nil {
		return err
	}
	mode, exist := ProviderTypeMapping[strings.ToLower(tp)]
	if !exist {
		return errors.New("invalid tun stack")
	}
	*e = mode
	return nil
}

// MarshalYAML serialize ProviderType with yaml
func (e ProviderType) MarshalYAML() (any, error) {
	return e.String(), nil
}

func (st ProviderType) String() string {
	switch st {
	case Proxy:
		return "gvisor"
	case Rule:
		return "lwip"
	default:
		return ""
	}
}

type Provider interface {
	Name() string
	Type() ProviderType
	Initial() error
}
