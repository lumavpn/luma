package inner

import (
	"errors"
	"net"
	"net/netip"
	"strconv"

	"github.com/lumavpn/luma/adapter"
	C "github.com/lumavpn/luma/common"
	N "github.com/lumavpn/luma/common/net"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/proto"
)

var tunnel adapter.TransportHandler

func New(t adapter.TransportHandler) {
	tunnel = t
}

func HandleTcp(address string, proxy string) (conn net.Conn, err error) {
	if tunnel == nil {
		return nil, errors.New("tcp uninitialized")
	}
	// executor Parsed
	conn1, conn2 := N.Pipe()

	metadata := &M.Metadata{}
	metadata.Network = M.TCP
	metadata.Type = proto.Proto_Inner
	metadata.DNSMode = C.DNSNormal
	metadata.Process = C.LumaName
	if proxy != "" {
		metadata.SpecialProxy = proxy
	}
	if h, port, err := net.SplitHostPort(address); err == nil {
		if port, err := strconv.ParseUint(port, 10, 16); err == nil {
			metadata.DstPort = uint16(port)
		}
		if ip, err := netip.ParseAddr(h); err == nil {
			metadata.DstIP = ip
		} else {
			metadata.Host = h
		}
	}

	go tunnel.HandleTCPConn(conn2, metadata)
	return conn1, nil
}
