package network

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/lumavpn/luma/common/pool"
	M "github.com/lumavpn/luma/metadata"
)

type AbstractConn interface {
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

type PacketReader interface {
	ReadPacket(buffer *pool.Buffer) (destination M.Socksaddr, err error)
}

type TimeoutPacketReader interface {
	PacketReader
	SetReadDeadline(t time.Time) error
}

type NetPacketReader interface {
	PacketReader
	ReadFrom(p []byte) (n int, addr net.Addr, err error)
}

type NetPacketWriter interface {
	PacketWriter
	WriteTo(p []byte, addr net.Addr) (n int, err error)
}

type PacketWriter interface {
	WritePacket(buffer *pool.Buffer, destination M.Socksaddr) error
}

type PacketConn interface {
	PacketReader
	PacketWriter

	Close() error
	LocalAddr() net.Addr
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

type ExtendedReader interface {
	io.Reader
	ReadBuffer(buffer *pool.Buffer) error
}

type ExtendedWriter interface {
	io.Writer
	WriteBuffer(buffer *pool.Buffer) error
}

type ExtendedConn interface {
	ExtendedReader
	ExtendedWriter
	net.Conn
}

type NetPacketConn interface {
	PacketConn
	NetPacketReader
	NetPacketWriter
}

type BindPacketConn interface {
	NetPacketConn
	net.Conn
}

type UDPHandler interface {
	NewPacket(ctx context.Context, conn PacketConn, buffer *pool.Buffer, metadata M.Metadata) error
}

type CachedReader interface {
	ReadCached() *pool.Buffer
}

type CachedPacketReader interface {
	ReadCachedPacket() *PacketBuffer
}

type PacketBuffer struct {
	Buffer      *pool.Buffer
	Destination M.Socksaddr
}

type WithUpstreamReader interface {
	UpstreamReader() any
}

type WithUpstreamWriter interface {
	UpstreamWriter() any
}

type ReaderWithUpstream interface {
	ReaderReplaceable() bool
}

type WriterWithUpstream interface {
	WriterReplaceable() bool
}
