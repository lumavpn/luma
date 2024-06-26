package tun

import "io"

const Driver = "tun"

type Tun interface {
	io.ReadWriter
	Close() error
}

type Options struct {
	Name           string
	MTU            uint32
	FileDescriptor int
}

func (t *TUN) Type() string {
	return Driver
}

var _ Device = (*TUN)(nil)
