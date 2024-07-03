//go:build !with_gvisor

package stack

import "github.com/lumavpn/luma/common/errors"

const WithGVisor = false

var ErrGVisorNotIncluded = errors.New(`gVisor is not included in this build, rebuild with -tags with_gvisor`)

func NewGVisor(
	cfg *Config,
) (Stack, error) {
	return nil, ErrGVisorNotIncluded
}

func NewMixed(
	cfg *Config,
) (Stack, error) {
	return nil, ErrGVisorNotIncluded
}
