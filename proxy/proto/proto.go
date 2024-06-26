package proto

import (
	"errors"
	"strings"
)

func EncodeProto(s string) (Proto, error) {
	pr, ok := Proto_value[strings.ToUpper(s)]
	if !ok {
		return Proto_PROTOCOL_UNSET, errors.New("Unknown proxy protocol")
	}

	return Proto(pr), nil
}
