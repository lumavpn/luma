package control

import (
	"syscall"

	"github.com/lumavpn/luma/common/errors"
)

func Conn(conn syscall.Conn, block func(fd uintptr) error) error {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return err
	}
	return Raw(rawConn, block)
}

func Raw(rawConn syscall.RawConn, block func(fd uintptr) error) error {
	var innerErr error
	err := rawConn.Control(func(fd uintptr) {
		innerErr = block(fd)
	})
	return errors.Errors(innerErr, err)
}
