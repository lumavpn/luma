package metadata

import (
	"net"
	"net/netip"
	"strconv"

	"github.com/lumavpn/luma/proxy/proto"
)

// Metadata contains metadata of transport protocol sessions.
type Metadata struct {
	Network Network     `json:"network"`
	SrcIP   netip.Addr  `json:"sourceIP"`
	DstIP   netip.Addr  `json:"destinationIP"`
	SrcPort uint16      `json:"sourcePort"`
	DstPort uint16      `json:"destinationPort"`
	Type    proto.Proto `json:"proto"`
	Host    string      `json:"host"`

	InName string     `json:"inboundName"`
	InUser string     `json:"inboundUser"`
	InIP   netip.Addr `json:"inboundIP"`
	InPort uint16     `json:"inboundPort,string"`

	Source      Socksaddr
	Destination Socksaddr
}

func (m *Metadata) SetRemoteAddress(rawAddress string) error {
	host, port, err := net.SplitHostPort(rawAddress)
	if err != nil {
		return err
	}

	var uint16Port uint16
	if port, err := strconv.ParseUint(port, 10, 16); err == nil {
		uint16Port = uint16(port)
	}

	if ip, err := netip.ParseAddr(host); err != nil {
		m.Host = host
		m.DstIP = netip.Addr{}
	} else {
		m.Host = ""
		m.DstIP = ip.Unmap()
	}
	m.DstPort = uint16Port

	return nil
}
