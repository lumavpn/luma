//go:build !windows

package tun

import (
	"github.com/lumavpn/luma/stack"
)

func tunNew(options stack.Options) (stack.Tun, error) {
	return stack.New(options)
}
