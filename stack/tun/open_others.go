//go:build !(linux && amd64) && !(linux && arm64) && !windows

package tun

func open(fd int, mtu uint32, offset int) (Device, error) {
	return nil, nil
}
