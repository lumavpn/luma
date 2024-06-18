//go:build !android

package tun

import (
	"github.com/lumavpn/luma/stack"
)

func (l *Listener) buildAndroidRules(tunOptions *stack.Options) error {
	return nil
}
func (l *Listener) openAndroidHotspot(tunOptions stack.Options) {}
