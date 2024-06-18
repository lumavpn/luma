package pool

import (
	"bytes"
	"sync"
)

var bufferPool = sync.Pool{New: func() any { return &bytes.Buffer{} }}

func GetBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

func PutBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bufferPool.Put(buf)
}

type Buffer = bytes.Buffer

func NewSize(n int) *bytes.Buffer {
	b := new(bytes.Buffer)
	b.Grow(n)
	return b
}
