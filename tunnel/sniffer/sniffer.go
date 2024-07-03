package sniffer

import M "github.com/lumavpn/luma/metadata"

type Sniffer interface {
	SupportNetwork() M.Network
	// SniffData must not change input bytes
	SniffData(bytes []byte) (string, error)
	Protocol() string
	SupportPort(port uint16) bool
}

const (
	TLS Type = iota
	HTTP
	QUIC
)

var (
	List = []Type{TLS, HTTP, QUIC}
)

type Type int

func (rt Type) String() string {
	switch rt {
	case TLS:
		return "TLS"
	case HTTP:
		return "HTTP"
	case QUIC:
		return "QUIC"
	default:
		return "Unknown"
	}
}
