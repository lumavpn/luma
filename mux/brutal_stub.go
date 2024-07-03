//go:build !linux

package mux

import (
	"net"

	E "github.com/lumavpn/luma/common/errors"
)

const BrutalAvailable = false

func SetBrutalOptions(conn net.Conn, sendBPS uint64) error {
	return E.New("TCP Brutal is only supported on Linux")
}
