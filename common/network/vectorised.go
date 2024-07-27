package network

import (
	M "github.com/lumavpn/luma/common/metadata"
	"github.com/sagernet/sing/common/buf"
)

type VectorisedWriter interface {
	WriteVectorised(buffers []*buf.Buffer) error
}

type VectorisedPacketWriter interface {
	WriteVectorisedPacket(buffers []*buf.Buffer, destination M.Socksaddr) error
}
