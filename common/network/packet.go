package network

import "net"

type WaitReadFrom interface {
	WaitReadFrom() (data []byte, put func(), addr net.Addr, err error)
}
