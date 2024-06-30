package conn

import (
	"github.com/lumavpn/luma/common/network"
)

func NeedHandshake(conn any) bool {
	if earlyConn, isEarlyConn := conn.(network.EarlyConn); isEarlyConn && earlyConn.NeedHandshake() {
		return true
	}
	return false
}
