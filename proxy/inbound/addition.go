package inbound

import (
	"net"

	C "github.com/lumavpn/luma/metadata"
	M "github.com/lumavpn/luma/metadata"
)

type Addition func(metadata *M.Metadata)

func ApplyAdditions(metadata *M.Metadata, additions ...Addition) {
	for _, addition := range additions {
		addition(metadata)
	}
}

func WithInName(name string) Addition {
	return func(metadata *M.Metadata) {
		metadata.InName = name
	}
}

func WithInUser(user string) Addition {
	return func(metadata *M.Metadata) {
		metadata.InUser = user
	}
}

func WithSpecialRules(specialRules string) Addition {
	return func(metadata *C.Metadata) {
		metadata.SpecialRules = specialRules
	}
}

func WithSpecialProxy(specialProxy string) Addition {
	return func(metadata *C.Metadata) {
		metadata.SpecialProxy = specialProxy
	}
}

func WithDstAddr(addr net.Addr) Addition {
	return func(metadata *C.Metadata) {
		_ = metadata.SetRemoteAddr(addr)
	}
}

func WithSrcAddr(addr net.Addr) Addition {
	return func(metadata *M.Metadata) {
		m := C.Metadata{}
		if err := m.SetRemoteAddr(addr); err == nil {
			metadata.SrcIP = m.DstIP
			metadata.SrcPort = m.DstPort
		}
	}
}

func WithInAddr(addr net.Addr) Addition {
	return func(metadata *M.Metadata) {
		m := C.Metadata{}
		if err := m.SetRemoteAddr(addr); err == nil {
			metadata.InIP = m.DstIP
			metadata.InPort = m.DstPort
		}
	}
}

func WithDSCP(dscp uint8) Addition {
	return func(metadata *M.Metadata) {
		metadata.DSCP = dscp
	}
}
