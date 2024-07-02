package inbound

import (
	"net"

	M "github.com/lumavpn/luma/metadata"
)

type Option func(metadata *M.Metadata)

func WithOptions(metadata *M.Metadata, optons ...Option) {
	for _, option := range optons {
		option(metadata)
	}
}

func WithDstAddr(addr net.Addr) Option {
	return func(metadata *M.Metadata) {
		_ = metadata.SetRemoteAddress(addr.String())
	}
}

func WithSrcAddr(addr net.Addr) Option {
	return func(metadata *M.Metadata) {
		m := M.Metadata{}
		if err := m.SetRemoteAddress(addr.String()); err == nil {
			metadata.SrcIP = m.DstIP
			metadata.SrcPort = m.DstPort
		}
	}
}

func WithInAddr(addr net.Addr) Option {
	return func(metadata *M.Metadata) {
		m := M.Metadata{}
		if err := m.SetRemoteAddress(addr.String()); err == nil {
			metadata.InIP = m.DstIP
			metadata.InPort = m.DstPort
		}
	}
}

func WithInUser(user string) Option {
	return func(metadata *M.Metadata) {
		metadata.InUser = user
	}
}
