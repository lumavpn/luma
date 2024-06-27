//go:build !with_gvisor

package stack

import "errors"

const WithGVisor = false

var ErrGVisorNotIncluded = errors.New(`gVisor is not included in this build, rebuild with -tags with_gvisor`)

func NewGVisor(
	opts *Options,
) (Stack, error) {
	return nil, ErrGVisorNotIncluded
}

func NewMixed(
	opts *Options,
) (Stack, error) {
	return nil, ErrGVisorNotIncluded
}
