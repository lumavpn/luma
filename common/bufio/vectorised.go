package bufio

import (
	"io"
	"net"
	"syscall"

	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/pool"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/util"
)

func NewVectorisedWriter(writer io.Writer) N.VectorisedWriter {
	if vectorisedWriter, ok := CreateVectorisedWriter(N.UnwrapWriter(writer)); ok {
		return vectorisedWriter
	}
	return &BufferedVectorisedWriter{upstream: writer}
}

func CreateVectorisedWriter(writer any) (N.VectorisedWriter, bool) {
	switch w := writer.(type) {
	case N.VectorisedWriter:
		return w, true
	case *net.TCPConn:
		return &NetVectorisedWriterWrapper{w}, true
	case *net.UDPConn:
		return &NetVectorisedWriterWrapper{w}, true
	case *net.IPConn:
		return &NetVectorisedWriterWrapper{w}, true
	case *net.UnixConn:
		return &NetVectorisedWriterWrapper{w}, true
	case syscall.Conn:
		rawConn, err := w.SyscallConn()
		if err == nil {
			return &SyscallVectorisedWriter{upstream: writer, rawConn: rawConn}, true
		}
	case syscall.RawConn:
		return &SyscallVectorisedWriter{upstream: writer, rawConn: w}, true
	}
	return nil, false
}

func CreateVectorisedPacketWriter(writer any) (N.VectorisedPacketWriter, bool) {
	switch w := writer.(type) {
	case N.VectorisedPacketWriter:
		return w, true
	case syscall.Conn:
		rawConn, err := w.SyscallConn()
		if err == nil {
			return &SyscallVectorisedPacketWriter{upstream: writer, rawConn: rawConn}, true
		}
	case syscall.RawConn:
		return &SyscallVectorisedPacketWriter{upstream: writer, rawConn: w}, true
	}
	return nil, false
}

var _ N.VectorisedWriter = (*BufferedVectorisedWriter)(nil)

type BufferedVectorisedWriter struct {
	upstream io.Writer
}

func (w *BufferedVectorisedWriter) WriteVectorised(buffers []*pool.Buffer) error {
	defer pool.ReleaseMulti(buffers)
	bufferLen := pool.LenMulti(buffers)
	if bufferLen == 0 {
		return util.Error(w.upstream.Write(nil))
	} else if len(buffers) == 1 {
		return util.Error(w.upstream.Write(buffers[0].Bytes()))
	}
	var bufferBytes []byte
	if bufferLen > 65535 {
		bufferBytes = make([]byte, bufferLen)
	} else {
		buffer := pool.NewSize(bufferLen)
		defer buffer.Release()
		bufferBytes = buffer.FreeBytes()
	}
	pool.CopyMulti(bufferBytes, buffers)
	return util.Error(w.upstream.Write(bufferBytes))
}

func (w *BufferedVectorisedWriter) Upstream() any {
	return w.upstream
}

var _ N.VectorisedWriter = (*NetVectorisedWriterWrapper)(nil)

type NetVectorisedWriterWrapper struct {
	upstream io.Writer
}

func (w *NetVectorisedWriterWrapper) WriteVectorised(buffers []*pool.Buffer) error {
	defer pool.ReleaseMulti(buffers)
	netBuffers := net.Buffers(pool.ToSliceMulti(buffers))
	return util.Error(netBuffers.WriteTo(w.upstream))
}

func (w *NetVectorisedWriterWrapper) Upstream() any {
	return w.upstream
}

func (w *NetVectorisedWriterWrapper) WriterReplaceable() bool {
	return true
}

var _ N.VectorisedWriter = (*SyscallVectorisedWriter)(nil)

type SyscallVectorisedWriter struct {
	upstream any
	rawConn  syscall.RawConn
	syscallVectorisedWriterFields
}

func (w *SyscallVectorisedWriter) Upstream() any {
	return w.upstream
}

func (w *SyscallVectorisedWriter) WriterReplaceable() bool {
	return true
}

var _ N.VectorisedPacketWriter = (*SyscallVectorisedPacketWriter)(nil)

type SyscallVectorisedPacketWriter struct {
	upstream any
	rawConn  syscall.RawConn
	syscallVectorisedWriterFields
}

func (w *SyscallVectorisedPacketWriter) Upstream() any {
	return w.upstream
}

var _ N.VectorisedPacketWriter = (*UnbindVectorisedPacketWriter)(nil)

type UnbindVectorisedPacketWriter struct {
	N.VectorisedWriter
}

func (w *UnbindVectorisedPacketWriter) WriteVectorisedPacket(buffers []*pool.Buffer, _ M.Socksaddr) error {
	return w.WriteVectorised(buffers)
}
