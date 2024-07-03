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

var LumaName = "luma"

// Metadata contains metadata of transport protocol sessions.
type Metadata struct {
	Network  Network    `json:"network"`
	SrcIP    netip.Addr `json:"sourceIP"`
	DstIP    netip.Addr `json:"destinationIP"`
	DstGeoIP []string   `json:"destinationGeoIP"` // can be nil if never queried, empty slice if got no result
	DstIPASN string     `json:"destinationIPASN"`

	SrcPort uint16 `json:"sourcePort"`
	MidPort uint16 `json:"dialerPort"`
	DstPort uint16 `json:"destinationPort"`

	Type   proto.Proto `json:"proto"`
	Host   string      `json:"host"`
	InName string      `json:"inboundName"`
	InUser string      `json:"inboundUser"`
	InIP   netip.Addr  `json:"inboundIP"`
	InPort uint16      `json:"inboundPort,string"` // `,string` is used to compatible with old version json output

	DNSMode      C.DNSMode `json:"dnsMode"`
	Uid          uint32    `json:"uid"`
	Process      string    `json:"process"`
	ProcessPath  string    `json:"processPath"`
	SpecialProxy string    `json:"specialProxy"`
	SpecialRules string    `json:"specialRules"`
	RemoteDst    string    `json:"remoteDestination"`
	DSCP         uint8     `json:"dscp"`

	RawSrcAddr net.Addr `json:"-"`
	RawDstAddr net.Addr `json:"-"`

	// Only domain rule
	SniffHost string `json:"sniffHost"`
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

func (m *Metadata) RuleHost() string {
	if len(m.SniffHost) == 0 {
		return m.Host
	} else {
		return m.SniffHost
	}
}

func (m *Metadata) JSON() string {
	b, _ := json.Marshal(m)
	return string(b)
}

// Pure is used to solve unexpected behavior
// when dialing proxy connection in DNSMapping mode.
func (m *Metadata) Pure() *Metadata {

	if (m.DNSMode == C.DNSMapping || m.DNSMode == C.DNSHosts) && m.DstIP.IsValid() {
		copyM := *m
		copyM.Host = ""
		return &copyM
	}

	return m
}

func (m *Metadata) SourceDetail() string {
	if m.Type == proto.Proto_Inner {
		return fmt.Sprintf("%s", LumaName)
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

func (m *Metadata) SourceValid() bool {
	return m.SrcPort != 0 && m.SrcIP.IsValid()
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

func (m *Metadata) Valid() bool {
	return m.Host != "" || m.DstIP.IsValid()
}

func (m *Metadata) Resolved() bool {
	return m.DstIP.IsValid()
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

func (m *Metadata) Addr() net.Addr {
	return &Addr{metadata: m}
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
	if addr, ok := addr.(interface{ AddrPort() netip.AddrPort }); ok { // *net.TCPAddr, *net.UDPAddr, M.Socksaddr
		if addrPort := addr.AddrPort(); addrPort.Port() != 0 {
			m.DstPort = addrPort.Port()
			if addrPort.IsValid() { // sing's M.Socksaddr maybe return an invalid AddrPort if it's a DomainName
				m.DstIP = addrPort.Addr().Unmap()
				return nil
			} else {
				if addr, ok := addr.(interface{ AddrString() string }); ok { // must be sing's M.Socksaddr
					m.Host = addr.AddrString() // actually is M.Socksaddr.Fqdn
					return nil
				}
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
