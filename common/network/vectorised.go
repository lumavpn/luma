package network

import (
	M "github.com/lumavpn/luma/common/metadata"
	"github.com/lumavpn/luma/common/pool"
)

type VectorisedWriter interface {
	WriteVectorised(buffers []*pool.Buffer) error
}

type VectorisedPacketWriter interface {
	WriteVectorisedPacket(buffers []*pool.Buffer, destination M.Socksaddr) error
}
