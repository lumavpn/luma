package inbound

import (
	"net"

	M "github.com/lumavpn/luma/metadata"
)

type Option func(meta *M.Metadata)

func WithOptions(metadata *M.Metadata, options ...Option) {
	for _, addition := range options {
		addition(metadata)
	}
}

func WithInName(name string) Option {
	return func(metadata *M.Metadata) {
		metadata.InName = name
	}
}

func WithInUser(user string) Option {
	return func(metadata *M.Metadata) {
		metadata.InUser = user
	}
}

func WithInAddr(addr net.Addr) Option {
	return func(metadata *M.Metadata) {
		m := M.Metadata{}
		if err := m.SetRemoteAddr(addr); err == nil {
			metadata.InIP = m.DstIP
			metadata.InPort = m.DstPort
		}
	}
}

func WithSrcAddr(addr net.Addr) Option {
	return func(metadata *M.Metadata) {
		m := M.Metadata{}
		if err := m.SetRemoteAddr(addr); err == nil {
			metadata.SrcIP = m.DstIP
			metadata.SrcPort = m.DstPort
		}
	}
}
