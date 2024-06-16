package protos

import (
	"errors"
	"strings"
)

func EncodeProto(s string) (Protocol, error) {
	pr, ok := Protocol_value[strings.ToUpper(s)]
	if !ok {
		return Protocol_PROTOCOL_UNSET, errors.New("Unknown proxy protocol")
	}

	return Protocol(pr), nil
}

func EncodeAdapterType(s string) (AdapterType, error) {
	at, ok := AdapterType_value[strings.Title(s)]
	if !ok {
		return AdapterType_AdapterType_Unset, errors.New("Unknown proxy adapter type")
	}

	return AdapterType(at), nil
}
