package udpnat

import (
	"context"
	"io"
	"net"
	"os"
	"time"

	"github.com/lumavpn/luma/common/cache"
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/pool"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/util"
)

type ErrorHandler interface {
	NewError(ctx context.Context, err error)
}

type Handler interface {
	N.UDPConnectionHandler
	ErrorHandler
}

type Service[K comparable] struct {
	nat     *cache.LruCache[K, *conn]
	handler Handler
}

func New[K comparable](maxAge int64, handler Handler) *Service[K] {
	return &Service[K]{
		nat: cache.New(
			cache.WithAge[K, *conn](maxAge),
			cache.WithUpdateAgeOnGet[K, *conn](),
			cache.WithEvict[K, *conn](func(key K, conn *conn) {
				conn.Close()
			}),
		),
		handler: handler,
	}
}

func (s *Service[T]) WriteIsThreadUnsafe() {
}

func (s *Service[T]) NewPacketDirect(ctx context.Context, key T, conn N.PacketConn, buffer *pool.Buffer, metadata M.Metadata) {
	s.NewContextPacket(ctx, key, buffer, metadata, func(natConn N.PacketConn) (context.Context, N.PacketWriter) {
		return ctx, &DirectBackWriter{conn, natConn}
	})
}

type DirectBackWriter struct {
	Source N.PacketConn
	Nat    N.PacketConn
}

func (w *DirectBackWriter) WritePacket(buffer *pool.Buffer, addr M.Socksaddr) error {
	return w.Source.WritePacket(buffer, M.SocksaddrFromNet(w.Nat.LocalAddr()))
}

func (w *DirectBackWriter) Upstream() any {
	return w.Source
}

func (s *Service[T]) NewPacket(ctx context.Context, key T, buffer *pool.Buffer, metadata M.Metadata, init func(natConn N.PacketConn) N.PacketWriter) {
	s.NewContextPacket(ctx, key, buffer, metadata, func(natConn N.PacketConn) (context.Context, N.PacketWriter) {
		return ctx, init(natConn)
	})
}

func (s *Service[T]) NewContextPacket(ctx context.Context, key T, buffer *pool.Buffer, metadata M.Metadata, init func(natConn N.PacketConn) (context.Context, N.PacketWriter)) {
	c, loaded := s.nat.LoadOrStore(key, func() *conn {
		c := &conn{
			data:       make(chan packet, 64),
			localAddr:  metadata.Source,
			remoteAddr: metadata.Destination,
		}
		c.ctx, c.cancel = context.WithCancelCause(ctx)
		return c
	})
	if !loaded {
		ctx, c.source = init(c)
		go func() {
			err := s.handler.NewPacketConnection(ctx, c, metadata)
			if err != nil {
				s.handler.NewError(ctx, err)
			}
			c.Close()
			s.nat.Delete(key)
		}()
	} else {
		c.localAddr = metadata.Source
	}
	if util.Done(c.ctx) {
		s.nat.Delete(key)
		if !util.Done(ctx) {
			s.NewContextPacket(ctx, key, buffer, metadata, init)
		}
		return
	}
	c.data <- packet{
		data:        buffer,
		destination: metadata.Destination,
	}
}

type packet struct {
	data        *pool.Buffer
	destination M.Socksaddr
}

var _ N.PacketConn = (*conn)(nil)

type conn struct {
	ctx             context.Context
	cancel          context.CancelCauseFunc
	data            chan packet
	localAddr       M.Socksaddr
	remoteAddr      M.Socksaddr
	source          N.PacketWriter
	readWaitOptions N.ReadWaitOptions
}

func (c *conn) ReadPacket(buffer *pool.Buffer) (addr M.Socksaddr, err error) {
	select {
	case p := <-c.data:
		_, err = buffer.ReadOnceFrom(p.data)
		p.data.Release()
		return p.destination, err
	case <-c.ctx.Done():
		return M.Socksaddr{}, io.ErrClosedPipe
	}
}

func (c *conn) WritePacket(buffer *pool.Buffer, destination M.Socksaddr) error {
	return c.source.WritePacket(buffer, destination)
}

func (c *conn) Close() error {
	select {
	case <-c.ctx.Done():
	default:
		c.cancel(net.ErrClosed)
	}
	if sourceCloser, sourceIsCloser := c.source.(io.Closer); sourceIsCloser {
		return sourceCloser.Close()
	}
	return nil
}

func (c *conn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *conn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *conn) SetDeadline(t time.Time) error {
	return os.ErrInvalid
}

func (c *conn) SetReadDeadline(t time.Time) error {
	return os.ErrInvalid
}

func (c *conn) SetWriteDeadline(t time.Time) error {
	return os.ErrInvalid
}

func (c *conn) NeedAdditionalReadDeadline() bool {
	return true
}

func (c *conn) Upstream() any {
	return c.source
}
