package bufio

import (
	"io"
	"net"
	"syscall"

	"github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/pool"
)

type BufferedVectorisedWriter struct {
	upstream io.Writer
}

func onError(_ any, err error) error {
	return err
}

func (w *BufferedVectorisedWriter) WriteVectorised(buffers []*pool.Buffer) error {
	defer pool.ReleaseMulti(buffers)
	bufferLen := pool.LenMulti(buffers)
	if bufferLen == 0 {
		return onError(w.upstream.Write(nil))
	} else if len(buffers) == 1 {
		return onError(w.upstream.Write(buffers[0].Bytes()))
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
	return onError(w.upstream.Write(bufferBytes))
}

func (w *BufferedVectorisedWriter) Upstream() any {
	return w.upstream
}

type NetVectorisedWriterWrapper struct {
	upstream io.Writer
}

func (w *NetVectorisedWriterWrapper) WriteVectorised(buffers []*pool.Buffer) error {
	defer pool.ReleaseMulti(buffers)
	netBuffers := net.Buffers(pool.ToSliceMulti(buffers))
	return onError(netBuffers.WriteTo(w.upstream))
}

func (w *NetVectorisedWriterWrapper) Upstream() any {
	return w.upstream
}

func (w *NetVectorisedWriterWrapper) WriterReplaceable() bool {
	return true
}

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

type SyscallVectorisedPacketWriter struct {
	upstream any
	rawConn  syscall.RawConn
	syscallVectorisedWriterFields
}

func (w *SyscallVectorisedPacketWriter) Upstream() any {
	return w.upstream
}

func CreateVectorisedWriter(writer any) (network.VectorisedWriter, bool) {
	switch w := writer.(type) {
	case network.VectorisedWriter:
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
