package canceler

import (
	"context"
	"net"
	"time"

	E "github.com/lumavpn/luma/common/errors"
	M "github.com/lumavpn/luma/common/metadata"
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/pool"
)

type TimeoutPacketConn struct {
	N.PacketConn
	timeout time.Duration
	cancel  context.CancelCauseFunc
	active  time.Time
}

func NewTimeoutPacketConn(ctx context.Context, conn N.PacketConn, timeout time.Duration) (context.Context, PacketConn) {
	ctx, cancel := context.WithCancelCause(ctx)
	return ctx, &TimeoutPacketConn{
		PacketConn: conn,
		timeout:    timeout,
		cancel:     cancel,
	}
}

func (c *TimeoutPacketConn) ReadPacket(buffer *pool.Buffer) (destination M.Socksaddr, err error) {
	for {
		err = c.PacketConn.SetReadDeadline(time.Now().Add(c.timeout))
		if err != nil {
			return
		}
		destination, err = c.PacketConn.ReadPacket(buffer)
		if err == nil {
			c.active = time.Now()
			return
		} else if E.IsTimeout(err) {
			if time.Since(c.active) > c.timeout {
				c.cancel(err)
				return
			}
		} else {
			return
		}
	}
}

func (c *TimeoutPacketConn) WritePacket(buffer *pool.Buffer, destination M.Socksaddr) error {
	err := c.PacketConn.WritePacket(buffer, destination)
	if err == nil {
		c.active = time.Now()
	}
	return err
}

func (c *TimeoutPacketConn) Timeout() time.Duration {
	return c.timeout
}

func (c *TimeoutPacketConn) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
	c.PacketConn.SetReadDeadline(time.Now())
}

func (c *TimeoutPacketConn) Close() error {
	c.cancel(net.ErrClosed)
	return c.PacketConn.Close()
}

func (c *TimeoutPacketConn) Upstream() any {
	return c.PacketConn
}
