package pool

import (
	"bytes"
	"sync"
	"sync/atomic"
)

var bufferPool = sync.Pool{New: func() any { return &bytes.Buffer{} }}

func GetBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

func PutBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bufferPool.Put(buf)
}

type Buffer struct {
	data     []byte
	start    int
	end      int
	capacity int
	refs     atomic.Int32
	managed  bool
}

func New() *Buffer {
	return &Buffer{
		data:     Get(BufferSize),
		capacity: BufferSize,
		managed:  true,
	}
}

func NewPacket() *Buffer {
	return &Buffer{
		data:     Get(UDPBufferSize),
		capacity: UDPBufferSize,
		managed:  true,
	}
}

func NewSize(size int) *Buffer {
	if size == 0 {
		return &Buffer{}
	} else if size > 65535 {
		return &Buffer{
			data:     make([]byte, size),
			capacity: size,
		}
	}
	return &Buffer{
		data:     Get(size),
		capacity: size,
		managed:  true,
	}
}

func As(data []byte) *Buffer {
	return &Buffer{
		data:     data,
		end:      len(data),
		capacity: len(data),
	}
}

func With(data []byte) *Buffer {
	return &Buffer{
		data:     data,
		capacity: len(data),
	}
}

func (b *Buffer) Byte(index int) byte {
	return b.data[b.start+index]
}

func (b *Buffer) Bytes() []byte {
	return b.data[b.start:b.end]
}

func (b *Buffer) Start() int {
	return b.start
}

func (b *Buffer) Len() int {
	return b.end - b.start
}

func (b *Buffer) FreeBytes() []byte {
	return b.data[b.end:b.capacity]
}
