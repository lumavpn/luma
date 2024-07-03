package canceler

import (
	"context"
	"time"

	"github.com/lumavpn/luma/common/buf"
	M "github.com/lumavpn/luma/common/metadata"
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/util"
)

type PacketConn interface {
	N.PacketConn
	Timeout() time.Duration
	SetTimeout(timeout time.Duration)
}

type TimerPacketConn struct {
	N.PacketConn
	instance *Instance
}

func NewPacketConn(ctx context.Context, conn N.PacketConn, timeout time.Duration) (context.Context, N.PacketConn) {
	if timeoutConn, isTimeoutConn := util.Cast[PacketConn](conn); isTimeoutConn {
		oldTimeout := timeoutConn.Timeout()
		if timeout < oldTimeout {
			timeoutConn.SetTimeout(timeout)
		}
		return ctx, conn
	}
	err := conn.SetReadDeadline(time.Time{})
	if err == nil {
		return NewTimeoutPacketConn(ctx, conn, timeout)
	}
	ctx, cancel := context.WithCancelCause(ctx)
	instance := New(ctx, cancel, timeout)
	return ctx, &TimerPacketConn{conn, instance}
}

func (c *TimerPacketConn) ReadPacket(buffer *buf.Buffer) (destination M.Socksaddr, err error) {
	destination, err = c.PacketConn.ReadPacket(buffer)
	if err == nil {
		c.instance.Update()
	}
	return
}

func (c *TimerPacketConn) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	err := c.PacketConn.WritePacket(buffer, destination)
	if err == nil {
		c.instance.Update()
	}
	return err
}

func (c *TimerPacketConn) Timeout() time.Duration {
	return c.instance.Timeout()
}

func (c *TimerPacketConn) SetTimeout(timeout time.Duration) {
	c.instance.SetTimeout(timeout)
}

func (c *TimerPacketConn) Close() error {
	return util.Close(
		c.PacketConn,
		c.instance,
	)
}

func (c *TimerPacketConn) Upstream() any {
	return c.PacketConn
}
