package network

import (
	"github.com/lumavpn/luma/util"
)

type HandshakeFailure interface {
	HandshakeFailure(err error) error
}

type HandshakeSuccess interface {
	HandshakeSuccess() error
}

func ReportHandshakeFailure(conn any, err error) error {
	if handshakeConn, isHandshakeConn := util.Cast[HandshakeFailure](conn); isHandshakeConn {
		return handshakeConn.HandshakeFailure(err)
	}
	return err
}

func ReportHandshakeSuccess(conn any) error {
	if handshakeConn, isHandshakeConn := util.Cast[HandshakeSuccess](conn); isHandshakeConn {
		return handshakeConn.HandshakeSuccess()
	}
	return nil
}
