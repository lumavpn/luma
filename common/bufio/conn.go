package bufio

import (
	"io"
	"net"

	M "github.com/lumavpn/luma/common/metadata"
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/pool"
	"github.com/lumavpn/luma/util"
)

func NewPacketConn(conn net.PacketConn) N.NetPacketConn {
	if udpConn, isUDPConn := conn.(*net.UDPConn); isUDPConn {
		return &ExtendedUDPConn{udpConn}
	} else if packetConn, isPacketConn := conn.(N.NetPacketConn); isPacketConn && !forceSTDIO {
		return packetConn
	} else {
		return &ExtendedPacketConn{conn}
	}
}

type ExtendedUDPConn struct {
	*net.UDPConn
}

func (w *ExtendedUDPConn) ReadPacket(buffer *pool.Buffer) (M.Socksaddr, error) {
	n, addr, err := w.ReadFromUDPAddrPort(buffer.FreeBytes())
	if err != nil {
		return M.Socksaddr{}, err
	}
	buffer.Truncate(n)
	return M.SocksaddrFromNetIP(addr).Unwrap(), nil
}

func (w *ExtendedUDPConn) WritePacket(buffer *pool.Buffer, destination M.Socksaddr) error {
	defer buffer.Release()
	if destination.IsFqdn() {
		udpAddr, err := net.ResolveUDPAddr("udp", destination.String())
		if err != nil {
			return err
		}
		return util.Error(w.UDPConn.WriteTo(buffer.Bytes(), udpAddr))
	}
	return util.Error(w.UDPConn.WriteToUDP(buffer.Bytes(), destination.UDPAddr()))
}

func (w *ExtendedUDPConn) Upstream() any {
	return w.UDPConn
}

type ExtendedPacketConn struct {
	net.PacketConn
}

func (w *ExtendedPacketConn) ReadPacket(buffer *pool.Buffer) (M.Socksaddr, error) {
	_, addr, err := buffer.ReadPacketFrom(w)
	if err != nil {
		return M.Socksaddr{}, err
	}
	return M.SocksaddrFromNet(addr).Unwrap(), err
}

func (w *ExtendedPacketConn) WritePacket(buffer *pool.Buffer, destination M.Socksaddr) error {
	defer buffer.Release()
	return util.Error(w.WriteTo(buffer.Bytes(), destination.UDPAddr()))
}

func (w *ExtendedPacketConn) Upstream() any {
	return w.PacketConn
}

type ExtendedReaderWrapper struct {
	io.Reader
}

func (r *ExtendedReaderWrapper) ReadBuffer(buffer *pool.Buffer) error {
	n, err := r.Read(buffer.FreeBytes())
	buffer.Truncate(n)
	if n > 0 && err == io.EOF {
		return nil
	}
	return err
}

func (r *ExtendedReaderWrapper) WriteTo(w io.Writer) (n int64, err error) {
	return Copy(w, r.Reader)
}

func (r *ExtendedReaderWrapper) Upstream() any {
	return r.Reader
}

func (r *ExtendedReaderWrapper) ReaderReplaceable() bool {
	return true
}

func NewExtendedReader(reader io.Reader) N.ExtendedReader {
	if forceSTDIO {
		if r, ok := reader.(*ExtendedReaderWrapper); ok {
			return r
		}
	} else {
		if r, ok := reader.(N.ExtendedReader); ok {
			return r
		}
	}
	return &ExtendedReaderWrapper{reader}
}

type ExtendedWriterWrapper struct {
	io.Writer
}

func (w *ExtendedWriterWrapper) WriteBuffer(buffer *pool.Buffer) error {
	defer buffer.Release()
	return util.Error(w.Write(buffer.Bytes()))
}

func (w *ExtendedWriterWrapper) ReadFrom(r io.Reader) (n int64, err error) {
	return Copy(w.Writer, r)
}

func (w *ExtendedWriterWrapper) Upstream() any {
	return w.Writer
}

func (w *ExtendedWriterWrapper) WriterReplaceable() bool {
	return true
}

func NewExtendedWriter(writer io.Writer) N.ExtendedWriter {
	if forceSTDIO {
		if w, ok := writer.(*ExtendedWriterWrapper); ok {
			return w
		}
	} else {
		if w, ok := writer.(N.ExtendedWriter); ok {
			return w
		}
	}
	return &ExtendedWriterWrapper{writer}
}

type ExtendedConnWrapper struct {
	net.Conn
	reader N.ExtendedReader
	writer N.ExtendedWriter
}

func (w *ExtendedConnWrapper) ReadBuffer(buffer *pool.Buffer) error {
	return w.reader.ReadBuffer(buffer)
}

func (w *ExtendedConnWrapper) WriteBuffer(buffer *pool.Buffer) error {
	return w.writer.WriteBuffer(buffer)
}

func (w *ExtendedConnWrapper) ReadFrom(r io.Reader) (n int64, err error) {
	return Copy(w.writer, r)
}

func (r *ExtendedConnWrapper) WriteTo(w io.Writer) (n int64, err error) {
	return Copy(w, r.reader)
}

func (w *ExtendedConnWrapper) UpstreamReader() any {
	return w.reader
}

func (w *ExtendedConnWrapper) ReaderReplaceable() bool {
	return true
}

func (w *ExtendedConnWrapper) UpstreamWriter() any {
	return w.writer
}

func (w *ExtendedConnWrapper) WriterReplaceable() bool {
	return true
}

func (w *ExtendedConnWrapper) Upstream() any {
	return w.Conn
}

func NewExtendedConn(conn net.Conn) N.ExtendedConn {
	if c, ok := conn.(N.ExtendedConn); ok {
		return c
	}
	return &ExtendedConnWrapper{
		Conn:   conn,
		reader: NewExtendedReader(conn),
		writer: NewExtendedWriter(conn),
	}
}