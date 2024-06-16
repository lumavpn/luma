package inbound

import (
	"net"

	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/protos"
	"github.com/lumavpn/luma/transport/socks5"
)

// NewSocket receive TCP inbound and return ConnContext
func NewSocket(target socks5.Addr, conn net.Conn, source protos.Protocol, options ...Option) (net.Conn, *M.Metadata) {
	metadata := parseSocksAddr(target)
	metadata.Network = M.TCP
	metadata.Type = source
	WithOptions(metadata, WithSrcAddr(conn.RemoteAddr()), WithInAddr(conn.LocalAddr()))
	WithOptions(metadata, options...)
	return conn, metadata
}