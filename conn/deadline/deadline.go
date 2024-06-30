package deadline

import (
	"net"

	"github.com/lumavpn/luma/common/bufio"
	"github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/conn"
)

func NewDeadlineConn(c net.Conn) conn.ExtendedConn {
	if IsConn(c) || IsConn(network.UnwrapReader(c)) {
		return bufio.NewExtendedConn(c)
	}
	return NewConn(c)
}
