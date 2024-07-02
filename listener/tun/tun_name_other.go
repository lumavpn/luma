//go:build !(darwin || linux)

package tun

import "os"

func getTunnelName(fd int32) (string, error) {
	return "", os.ErrInvalid
}
