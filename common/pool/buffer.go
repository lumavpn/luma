package pool

import (
	"bytes"
	"io"
	"log"
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

func (b *Buffer) Release() {
	if b == nil || !b.managed {
		return
	}
	if b.refs.Load() > 0 {
		return
	}
	Put(b.data)
	*b = Buffer{}
}

func (b *Buffer) Bytes() []byte {
	return b.data[b.start:b.end]
}

func (b *Buffer) Len() int {
	return b.end - b.start
}

func (b *Buffer) FreeBytes() []byte {
	return b.data[b.end:b.capacity]
}

// IsFull indicates whether or not this buffer has reached capacity
func (b *Buffer) IsFull() bool {
	return b.end == b.capacity
}

func (b *Buffer) Truncate(to int) {
	b.end = b.start + to
}

func (b *Buffer) Write(data []byte) (n int, err error) {
	if len(data) == 0 {
		return
	}
	if b.IsFull() {
		return 0, io.ErrShortBuffer
	}
	n = copy(b.data[b.end:b.capacity], data)
	b.end += n
	return
}

func (b *Buffer) WriteByte(d byte) error {
	if b.IsFull() {
		return io.ErrShortBuffer
	}
	b.data[b.end] = d
	b.end++
	return nil
}

func (b *Buffer) WriteString(s string) (n int, err error) {
	if len(s) == 0 {
		return
	}
	if b.IsFull() {
		return 0, io.ErrShortBuffer
	}
	n = copy(b.data[b.end:b.capacity], s)
	b.end += n
	return
}

func (b *Buffer) Extend(n int) []byte {
	end := b.end + n
	if end > b.capacity {
		log.Fatal("buffer overflow: capacity ", b.capacity, ",end ", b.end, ", need ", n)
	}
	ext := b.data[b.end:end]
	b.end = end
	return ext
}
