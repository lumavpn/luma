package rw

import (
	"github.com/lumavpn/luma/util"
)

type ReadCloser interface {
	CloseRead() error
}

type WriteCloser interface {
	CloseWrite() error
}

func CloseRead(reader any) error {
	if c, ok := util.Cast[ReadCloser](reader); ok {
		return c.CloseRead()
	}
	return nil
}

func CloseWrite(writer any) error {
	if c, ok := util.Cast[WriteCloser](writer); ok {
		return c.CloseWrite()
	}
	return nil
}
