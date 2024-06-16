package metadata

import (
	"net"
	"net/netip"
	"strconv"

	"github.com/lumavpn/luma/proxy/protos"
)

// Metadata represents a transport protocol session
type Metadata struct {
	Network Network    `json:"network"`
	SrcIP   netip.Addr `json:"sourceIP"`
	DstIP   netip.Addr `json:"destinationIP"`

	SrcPort uint16 `json:"sourcePort"`
	MidPort uint16 `json:"midPort"`
	DstPort uint16 `json:"destinationPort"`

	Host string `json:"host"`

	Type protos.Protocol `json:"proto"`

	RawSrcAddr net.Addr `json:"-"`
	RawDstAddr net.Addr `json:"-"`
}

func (m *Metadata) DestinationAddress() string {
	return net.JoinHostPort(m.DstIP.String(), strconv.FormatUint(uint64(m.DstPort), 10))
}

func (m *Metadata) SourceAddress() string {
	return net.JoinHostPort(m.SrcIP.String(), strconv.FormatUint(uint64(m.SrcPort), 10))
}

func (m *Metadata) AddrPort() netip.AddrPort {
	return netip.AddrPortFrom(m.DstIP.Unmap(), m.DstPort)
}

func (m *Metadata) UDPAddr() *net.UDPAddr {
	if m.Network != UDP || !m.DstIP.IsValid() {
		return nil
	}
	return net.UDPAddrFromAddrPort(m.AddrPort())
}
