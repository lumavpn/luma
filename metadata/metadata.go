package metadata

import (
	"fmt"
	"net"
	"net/netip"
	"strconv"

	"github.com/lumavpn/luma/dns"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/transport/socks5"
)

// Metadata contains metadata of transport protocol sessions.
type Metadata struct {
	Network Network `json:"network"`

	Proto proto.Proto

	SrcIP  netip.Addr `json:"sourceIP"`
	DstIP  netip.Addr `json:"destinationIP"`
	MidIP  net.IP     `json:"dialerIP"`
	InName string     `json:"inboundName"`
	InUser string     `json:"inboundUser"`
	InIP   netip.Addr `json:"inboundIP"`
	InPort uint16     `json:"inboundPort,string"`

	SpecialProxy string `json:"specialProxy"`
	SpecialRules string `json:"specialRules"`
	SrcPort      uint16 `json:"sourcePort"`
	MidPort      uint16 `json:"dialerPort"`
	DstPort      uint16 `json:"destinationPort"`
	Host         string

	DNSMode dns.DNSMode `json:"dnsMode"`

	RawSrcAddr  net.Addr `json:"-"`
	RawDstAddr  net.Addr `json:"-"`
	Source      Socksaddr
	Destination Socksaddr
}

func (m *Metadata) DestinationAddress() string {
	return net.JoinHostPort(m.DstIP.String(), strconv.FormatUint(uint64(m.DstPort), 10))
}

func (m *Metadata) SourceAddress() string {
	return net.JoinHostPort(m.SrcIP.String(), strconv.FormatUint(uint64(m.SrcPort), 10))
}

func (m *Metadata) SourceAddrPort() netip.AddrPort {
	return netip.AddrPortFrom(m.SrcIP.Unmap(), m.SrcPort)
}

func (m *Metadata) FiveTuple() string {
	return fmt.Sprintf("[%s] %s -> %s", m.Network.String(), m.Source.String(), m.Destination.String())
}

func (m *Metadata) Pure() *Metadata {
	if (m.DNSMode == dns.DNSMapping || m.DNSMode == dns.DNSHosts) && m.DstIP.IsValid() {
		copyM := *m
		copyM.Host = ""
		return &copyM
	}
	return m
}

func (m *Metadata) Addr() net.Addr {
	return &Addr{metadata: m}
}

func (m *Metadata) AddrType() int {
	switch true {
	case m.Host != "" || !m.DstIP.IsValid():
		return socks5.AtypDomainName
	case m.DstIP.Is4():
		return socks5.AtypIPv4
	default:
		return socks5.AtypIPv6
	}
}

func (m *Metadata) Resolved() bool {
	return m.DstIP.IsValid()
}

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

func (m *Metadata) Valid() bool {
	return m.Host != "" || m.DstIP.IsValid()
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

// Addr implements the net.Addr interface.
type Addr struct {
	metadata *Metadata
}

func (a *Addr) Metadata() *Metadata {
	return a.metadata
}

func (a *Addr) Network() string {
	return a.metadata.Network.String()
}

func (a *Addr) String() string {
	return a.metadata.DestinationAddress()
}
