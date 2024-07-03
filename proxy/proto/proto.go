package proto

import (
	"errors"
	"strings"
)

func EncodeProto(s string) (Proto, error) {
	pr, ok := Proto_value[strings.Title(s)]
	if !ok {
		return Proto_Unset, errors.New("Unknown proxy protocol")
	}

	return Proto(pr), nil
}
