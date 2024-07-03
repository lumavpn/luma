package metadata

import (
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"strconv"

	C "github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/transport/socks5"
)

// Metadata contains metadata of transport protocol sessions.
type Metadata struct {
	Network  Network    `json:"network"`
	SrcIP    netip.Addr `json:"sourceIP"`
	DstIP    netip.Addr `json:"destinationIP"`
	DstGeoIP []string   `json:"destinationGeoIP"` // can be nil if never queried, empty slice if got no result
	DstIPASN string     `json:"destinationIPASN"`

	SrcPort uint16      `json:"sourcePort"`
	DstPort uint16      `json:"destinationPort"`
	Type    proto.Proto `json:"proto"`
	Host    string      `json:"host"`

	InName string     `json:"inboundName"`
	InUser string     `json:"inboundUser"`
	InIP   netip.Addr `json:"inboundIP"`
	InPort uint16     `json:"inboundPort,string"`

	DNSMode      C.DNSMode `json:"dnsMode"`
	Uid          uint32    `json:"uid"`
	Process      string    `json:"process"`
	ProcessPath  string    `json:"processPath"`
	SpecialProxy string    `json:"specialProxy"`
	SpecialRules string    `json:"specialRules"`
	RemoteDst    string    `json:"remoteDestination"`
	DSCP         uint8     `json:"dscp"`

	// Only domain rule
	SniffHost string `json:"sniffHost"`
}

func (m *Metadata) SetRemoteAddr(addr net.Addr) error {
	if addr == nil {
		return nil
	}
	return m.SetRemoteAddress(addr.String())
}

func (m *Metadata) RuleHost() string {
	if len(m.SniffHost) == 0 {
		return m.Host
	} else {
		return m.SniffHost
	}
}

func (m *Metadata) Pure() *Metadata {
	if (m.DNSMode == C.DNSMapping || m.DNSMode == C.DNSHosts) && m.DstIP.IsValid() {
		copyM := *m
		copyM.Host = ""
		return &copyM
	}
	return m
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

func (m *Metadata) SourceDetail() string {
	if m.Type == proto.Proto_Inner {
		return fmt.Sprintf("%s", C.LumaName)
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

func (m *Metadata) DestinationAddress() string {
	return net.JoinHostPort(m.DstIP.String(), strconv.FormatUint(uint64(m.DstPort), 10))
}

func (m *Metadata) RemoteAddress() string {
	return net.JoinHostPort(m.String(), strconv.FormatUint(uint64(m.DstPort), 10))
}

func (m *Metadata) SourceAddress() string {
	return net.JoinHostPort(m.SrcIP.String(), strconv.FormatUint(uint64(m.SrcPort), 10))
}

func (m *Metadata) SourceAddrPort() netip.AddrPort {
	return netip.AddrPortFrom(m.SrcIP.Unmap(), m.SrcPort)
}

func (m *Metadata) Valid() bool {
	return m.Host != "" || m.DstIP.IsValid()
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

func (m *Metadata) JSON() string {
	b, _ := json.Marshal(m)
	return string(b)
}

func (m *Metadata) String() string {
	if m.Host != "" {
		return m.Host
	} else if m.DstIP.IsValid() {
		return m.DstIP.String()
	} else {
		return ""
	}
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
