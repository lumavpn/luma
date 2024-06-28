package network

import (
	"github.com/lumavpn/luma/common/pool"
	M "github.com/lumavpn/luma/metadata"
)

type VectorisedWriter interface {
	WriteVectorised(buffers []*pool.Buffer) error
}

type VectorisedPacketWriter interface {
	WriteVectorisedPacket(buffers []*pool.Buffer, destination M.Socksaddr) error
}
