package inbound

import (
	"net"

	C "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/transport/socks5"
)

// NewHTTP receive normal http request and return HTTPContext
func NewHTTP(target socks5.Addr, srcConn net.Conn, conn net.Conn, additions ...Addition) (net.Conn, *C.Metadata) {
	metadata := parseSocksAddr(target)
	metadata.Network = C.TCP
	metadata.Type = proto.Proto_HTTP
	metadata.RawSrcAddr = srcConn.RemoteAddr()
	metadata.RawDstAddr = srcConn.LocalAddr()
	ApplyAdditions(metadata, WithSrcAddr(srcConn.RemoteAddr()), WithInAddr(srcConn.LocalAddr()))
	ApplyAdditions(metadata, additions...)
	return conn, metadata
}
