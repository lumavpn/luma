package bufio

import (
	M "github.com/lumavpn/luma/common/metadata"
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/pool"
)

var _ N.ReadWaiter = (*BindPacketReadWaiter)(nil)

type BindPacketReadWaiter struct {
	readWaiter N.PacketReadWaiter
}

func (w *BindPacketReadWaiter) InitializeReadWaiter(options N.ReadWaitOptions) (needCopy bool) {
	return w.readWaiter.InitializeReadWaiter(options)
}

func (w *BindPacketReadWaiter) WaitReadBuffer() (buffer *pool.Buffer, err error) {
	buffer, _, err = w.readWaiter.WaitReadPacket()
	return
}

var _ N.PacketReadWaiter = (*UnbindPacketReadWaiter)(nil)

type UnbindPacketReadWaiter struct {
	readWaiter N.ReadWaiter
	addr       M.Socksaddr
}

func (w *UnbindPacketReadWaiter) InitializeReadWaiter(options N.ReadWaitOptions) (needCopy bool) {
	return w.readWaiter.InitializeReadWaiter(options)
}

func (w *UnbindPacketReadWaiter) WaitReadPacket() (buffer *pool.Buffer, destination M.Socksaddr, err error) {
	buffer, err = w.readWaiter.WaitReadBuffer()
	if err != nil {
		return
	}
	destination = w.addr
	return
}
