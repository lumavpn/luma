package network

import (
	"github.com/lumavpn/luma/common/pool"
)

type VectorisedWriter interface {
	WriteVectorised(buffers []*pool.Buffer) error
}
