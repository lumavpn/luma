//go:build !(linux || windows || darwin)

package stack

import (
	"os"
)

func New(config Options) (Tun, error) {
	return nil, os.ErrInvalid
}
