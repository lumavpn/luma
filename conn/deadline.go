package conn

import (
	"net"

	"github.com/lumavpn/luma/common/bufio"
	"github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/network/deadline"
)

func NewDeadlineConn(conn net.Conn) network.ExtendedConn {
	if deadline.IsPipe(conn) || deadline.IsPipe(network.UnwrapReader(conn)) {
		return bufio.NewExtendedConn(conn) // was a *deadline.Conn
	}
	if deadline.IsConn(conn) || deadline.IsConn(network.UnwrapReader(conn)) {
		return bufio.NewExtendedConn(conn) // was a *deadline.Conn
	}
	return deadline.NewConn(conn)
}
