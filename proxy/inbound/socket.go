package inbound

import (
	"net"

	C "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/transport/socks5"
)

// NewSocket receive TCP inbound and return ConnContext
func NewSocket(target socks5.Addr, conn net.Conn, source proto.Proto, additions ...Addition) (net.Conn, *C.Metadata) {
	metadata := parseSocksAddr(target)
	metadata.Network = C.TCP
	metadata.Type = source
	ApplyAdditions(metadata, WithSrcAddr(conn.RemoteAddr()), WithInAddr(conn.LocalAddr()))
	ApplyAdditions(metadata, additions...)
	return conn, metadata
}
