package metadata

import (
	"net"
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

	Source      Socksaddr
	Destination Socksaddr
}
