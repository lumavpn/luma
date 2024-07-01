package pool

import (
	"bytes"
	"crypto/rand"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"

	"github.com/lumavpn/luma/common/errors"
	"github.com/lumavpn/luma/util"
)

const debugEnabled = false

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

func (b *Buffer) FreeLen() int {
	return b.capacity - b.end
}

func (b *Buffer) IsEmpty() bool {
	return b.end-b.start == 0
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

func (b *Buffer) Resize(start, end int) {
	b.start = start
	b.end = b.start + end
}

func (b *Buffer) ReadFullFrom(r io.Reader, size int) (n int, err error) {
	if b.end+size > b.capacity {
		return 0, io.ErrShortBuffer
	}
	n, err = io.ReadFull(r, b.data[b.end:b.end+size])
	b.end += n
	return
}

func (b *Buffer) ReadFrom(reader io.Reader) (n int64, err error) {
	for {
		if b.IsFull() {
			return 0, io.ErrShortBuffer
		}
		var readN int
		readN, err = reader.Read(b.FreeBytes())
		b.end += readN
		n += int64(readN)
		if err != nil {
			if errors.IsMulti(err, io.EOF) {
				err = nil
			}
			return
		}
	}
}

func (b *Buffer) ReadPacketFrom(r net.PacketConn) (int64, net.Addr, error) {
	if b.IsFull() {
		return 0, nil, io.ErrShortBuffer
	}
	n, addr, err := r.ReadFrom(b.FreeBytes())
	b.end += n
	return int64(n), addr, err
}

func (b *Buffer) Reserve(n int) {
	if n > b.capacity {
		log.Fatal("buffer overflow: capacity ", b.capacity, ", need ", n)
	}
	b.capacity -= n
}

func (b *Buffer) OverCap(n int) {
	if b.capacity+n > len(b.data) {
		log.Fatal("buffer overflow: capacity ", len(b.data), ", need ", b.capacity+n)
	}
	b.capacity += n
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

func (b *Buffer) IncRef() {
	b.refs.Add(1)
}

func (b *Buffer) DecRef() {
	b.refs.Add(-1)
}

func (b *Buffer) Leak() {
	if debugEnabled {
		if b == nil || !b.managed {
			return
		}
		refs := b.refs.Load()
		if refs == 0 {
			panic("leaking buffer")
		} else {
			log.Fatal("leaking buffer with ", refs, " references")
		}
	} else {
		b.Release()
	}
}

func (b *Buffer) ReadOnceFrom(r io.Reader) (int, error) {
	if b.IsFull() {
		return 0, io.ErrShortBuffer
	}
	n, err := r.Read(b.FreeBytes())
	b.end += n
	return n, err
}

func (b *Buffer) Advance(from int) {
	b.start += from
}

func (b *Buffer) ExtendHeader(n int) []byte {
	if b.start < n {
		panic(util.ToString("buffer overflow: capacity ", b.capacity, ",start ", b.start, ", need ", n))
	}
	b.start -= n
	return b.data[b.start : b.start+n]
}

func (b *Buffer) SetByte(index int, value byte) {
	b.data[b.start+index] = value
}

func (b *Buffer) Extend(n int) []byte {
	end := b.end + n
	if end > b.capacity {
		panic(util.ToString("buffer overflow: capacity ", b.capacity, ",end ", b.end, ", need ", n))
	}
	ext := b.data[b.end:end]
	b.end = end
	return ext
}

func (b *Buffer) WriteRandom(size int) []byte {
	buffer := b.Extend(size)
	util.Must1(io.ReadFull(rand.Reader, buffer))
	return buffer
}

func (b *Buffer) WriteByte(d byte) error {
	if b.IsFull() {
		return io.ErrShortBuffer
	}
	b.data[b.end] = d
	b.end++
	return nil
}

func (b *Buffer) WriteRune(s rune) (int, error) {
	return b.Write([]byte{byte(s)})
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
