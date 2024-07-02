package udpnat

import (
	"io"

	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/pool"
	M "github.com/lumavpn/luma/metadata"
)

var _ N.PacketReadWaiter = (*conn)(nil)

func (c *conn) InitializeReadWaiter(options N.ReadWaitOptions) (needCopy bool) {
	c.readWaitOptions = options
	return false
}

func (c *conn) WaitReadPacket() (buffer *pool.Buffer, destination M.Socksaddr, err error) {
	select {
	case p := <-c.data:
		if c.readWaitOptions.NeedHeadroom() {
			buffer = c.readWaitOptions.NewPacketBuffer()
			_, err = buffer.Write(p.data.Bytes())
			if err != nil {
				buffer.Release()
				return
			}
			c.readWaitOptions.PostReturn(buffer)
			p.data.Release()
		} else {
			buffer = p.data
		}
		destination = p.destination
		return
	case <-c.ctx.Done():
		return nil, M.Socksaddr{}, io.ErrClosedPipe
	}
}
