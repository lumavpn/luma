package mux

import (
	"io"
	"net"
	"strings"

	"github.com/hashicorp/yamux"
	E "github.com/lumavpn/luma/common/errors"
)

type wrapStream struct {
	net.Conn
}

func (w *wrapStream) Read(p []byte) (n int, err error) {
	n, err = w.Conn.Read(p)
	err = wrapError(err)
	return
}

func (w *wrapStream) Write(p []byte) (n int, err error) {
	n, err = w.Conn.Write(p)
	err = wrapError(err)
	return
}

func (w *wrapStream) Upstream() any {
	return w.Conn
}

func wrapError(err error) error {
	switch err {
	case yamux.ErrStreamClosed:
		return io.EOF
	default:
		return err
	}
}

func Contains(err error, msgList ...string) bool {
	for _, msg := range msgList {
		if strings.Contains(err.Error(), msg) {
			return true
		}
	}
	return false
}

func WrapH2(err error) error {
	if err == nil {
		return nil
	}
	err = E.Unwrap(err)
	if err == io.ErrUnexpectedEOF {
		return io.EOF
	}
	if Contains(err, "client disconnected", "body closed by handler", "response body closed", "; CANCEL") {
		return net.ErrClosed
	}
	return err
}
