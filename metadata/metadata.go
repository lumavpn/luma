package metadata

import (
	"fmt"
	"net"
	"net/netip"
	"strconv"

	"github.com/lumavpn/luma/common"
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

	InName string     `json:"inboundName"`
	InUser string     `json:"inboundUser"`
	InIP   netip.Addr `json:"inboundIP"`
	InPort uint16     `json:"inboundPort,string"`

	Uid         uint32 `json:"uid"`
	Process     string `json:"process"`
	ProcessPath string `json:"processPath"`

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

func (m *Metadata) SourceDetail() string {
	if m.Type == protos.Protocol_INNER {
		return fmt.Sprintf("%s", common.LumaName)
	}
	switch {
	case m.Process != "" && m.Uid != 0:
		return fmt.Sprintf("%s(%s, uid=%d)", m.SourceAddress(), m.Process, m.Uid)
	case m.Uid != 0:
		return fmt.Sprintf("%s(uid=%d)", m.SourceAddress(), m.Uid)
	case m.Process != "":
		return fmt.Sprintf("%s(%s)", m.SourceAddress(), m.Process)
	default:
		return fmt.Sprintf("%s", m.SourceAddress())
	}
}

func (m *Metadata) Valid() bool {
	return m.Host != "" || m.DstIP.IsValid()
}

func (m *Metadata) UDPAddr() *net.UDPAddr {
	if m.Network != UDP || !m.DstIP.IsValid() {
		return nil
	}
	return net.UDPAddrFromAddrPort(m.AddrPort())
}

// SetRemoteAddr updates the destination address to be the same as the given net.Addr
func (m *Metadata) SetRemoteAddr(addr net.Addr) error {
	if addr == nil {
		return nil
	}
	if rawAddr, ok := addr.(interface{ RawAddr() net.Addr }); ok {
		if rawAddr := rawAddr.RawAddr(); rawAddr != nil {
			if err := m.SetRemoteAddr(rawAddr); err == nil {
				return nil
			}
		}
	}
	if addr, ok := addr.(interface{ AddrPort() netip.AddrPort }); ok {
		if addrPort := addr.AddrPort(); addrPort.Port() != 0 {
			m.DstPort = addrPort.Port()
			if addrPort.IsValid() {
				m.DstIP = addrPort.Addr().Unmap()
				return nil
			}
		}
	}
	return m.SetRemoteAddress(addr.String())
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
