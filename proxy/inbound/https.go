package inbound

import (
	"net"
	"net/http"

	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/proto"
)

// NewHTTPS receive CONNECT request and return ConnContext
func NewHTTPS(request *http.Request, conn net.Conn, additions ...Addition) (net.Conn, *M.Metadata) {
	metadata := parseHTTPAddr(request)
	metadata.Type = proto.Proto_HTTPS
	ApplyAdditions(metadata, WithSrcAddr(conn.RemoteAddr()), WithInAddr(conn.LocalAddr()))
	ApplyAdditions(metadata, additions...)
	return conn, metadata
}
