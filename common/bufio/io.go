package bufio

import (
	"github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/pool"
)

func WriteVectorised(writer network.VectorisedWriter, data [][]byte) (n int, err error) {
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
