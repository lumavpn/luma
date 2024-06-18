package rw

import (
	"io"

	"github.com/lumavpn/luma/common/pool"
)

var Discard io.Writer = discard{}

type discard struct{}

var _ io.ReaderFrom = discard{}

func (discard) Write(p []byte) (int, error) {
	return len(p), nil
}

func (discard) WriteString(s string) (int, error) {
	return len(s), nil
}

func (discard) ReadFrom(r io.Reader) (n int64, err error) {
	buffer := pool.Get(pool.RelayBufferSize)
	readSize := 0
	for {
		readSize, err = r.Read(buffer)
		n += int64(readSize)
		if err != nil {
			pool.Put(buffer)
			if err == io.EOF {
				return n, nil
			}
			return
		}
	}
}
