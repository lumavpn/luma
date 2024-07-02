package network

import (
	"github.com/lumavpn/luma/common/pool"
	M "github.com/lumavpn/luma/metadata"
)

type ReadWaitable interface {
	InitializeReadWaiter(options ReadWaitOptions) (needCopy bool)
}

type ReadWaitOptions struct {
	FrontHeadroom int
	RearHeadroom  int
	MTU           int
}

func (o ReadWaitOptions) NeedHeadroom() bool {
	return o.FrontHeadroom > 0 || o.RearHeadroom > 0
}

func (o ReadWaitOptions) NewBuffer() *pool.Buffer {
	return o.newBuffer(pool.RelayBufferSize)
}

func (o ReadWaitOptions) NewPacketBuffer() *pool.Buffer {
	return o.newBuffer(pool.UDPBufferSize)
}

func (o ReadWaitOptions) newBuffer(defaultBufferSize int) *pool.Buffer {
	var bufferSize int
	if o.MTU > 0 {
		bufferSize = o.MTU + o.FrontHeadroom + o.RearHeadroom
	} else {
		bufferSize = defaultBufferSize
	}
	buffer := pool.NewSize(bufferSize)
	if o.FrontHeadroom > 0 {
		buffer.Resize(o.FrontHeadroom, 0)
	}
	if o.RearHeadroom > 0 {
		buffer.Reserve(o.RearHeadroom)
	}
	return buffer
}

func (o ReadWaitOptions) PostReturn(buffer *pool.Buffer) {
	if o.RearHeadroom > 0 {
		buffer.OverCap(o.RearHeadroom)
	}
}

type ReadWaiter interface {
	ReadWaitable
	WaitReadBuffer() (buffer *pool.Buffer, err error)
}

type ReadWaitCreator interface {
	CreateReadWaiter() (ReadWaiter, bool)
}

type PacketReadWaiter interface {
	ReadWaitable
	WaitReadPacket() (buffer *pool.Buffer, destination M.Socksaddr, err error)
}

type PacketReadWaitCreator interface {
	CreateReadWaiter() (PacketReadWaiter, bool)
}
