package network

import (
	"github.com/lumavpn/luma/common/pool"
)

type VectorisedWriter interface {
	WriteVectorised(buffers []*pool.Buffer) error
}

func WriteVectorised(writer VectorisedWriter, data [][]byte) (n int, err error) {
	var dataLen int
	buffers := make([]*pool.Buffer, 0, len(data))
	for _, p := range data {
		dataLen += len(p)
		buffers = append(buffers, pool.As(p))
	}
	err = writer.WriteVectorised(buffers)
	if err == nil {
		n = dataLen
	}
	return
}
