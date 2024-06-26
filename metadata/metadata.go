package metadata

import (
	"net"
	"strconv"
)

// Metadata contains metadata of transport protocol sessions.
type Metadata struct {
	Network Network `json:"network"`
	SrcIP   net.IP  `json:"sourceIP"`
	MidIP   net.IP  `json:"dialerIP"`
	DstIP   net.IP  `json:"destinationIP"`
	SrcPort uint16  `json:"sourcePort"`
	MidPort uint16  `json:"dialerPort"`
	DstPort uint16  `json:"destinationPort"`
}

func (m *Metadata) DestinationAddress() string {
	return net.JoinHostPort(m.DstIP.String(), strconv.FormatUint(uint64(m.DstPort), 10))
}

func (m *Metadata) SourceAddress() string {
	return net.JoinHostPort(m.SrcIP.String(), strconv.FormatUint(uint64(m.SrcPort), 10))
}
