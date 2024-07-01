//go:build !linux

package mux

import (
	"errors"
	"net"
)

const BrutalAvailable = false

func SetBrutalOptions(conn net.Conn, sendBPS uint64) error {
	return errors.New("TCP Brutal is only supported on Linux")
}
