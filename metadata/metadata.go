package metadata

import (
	"net"
	"net/netip"
	"strconv"
)

// Metadata represents a transport protocol session
type Metadata struct {
	Network Network    `json:"network"`
	SrcIP   netip.Addr `json:"sourceIP"`
	DstIP   netip.Addr `json:"destinationIP"`

	SrcPort uint16 `json:"sourcePort"`
	MidPort uint16 `json:"midPort"`
	DstPort uint16 `json:"destinationPort"`
}

func (m *Metadata) DestinationAddress() string {
	return net.JoinHostPort(m.DstIP.String(), strconv.FormatUint(uint64(m.DstPort), 10))
}
