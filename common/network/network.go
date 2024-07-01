package network

import (
	"github.com/lumavpn/luma/util"
)

func NeedHandshake(conn any) bool {
	if earlyConn, isEarlyConn := util.Cast[EarlyConn](conn); isEarlyConn && earlyConn.NeedHandshake() {
		return true
	}
	return false
}
