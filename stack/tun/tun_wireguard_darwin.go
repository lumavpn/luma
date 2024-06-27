//go:build with_wireguard && with_gvisor && darwin

package tun

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/lumavpn/luma/common/pool"
	"github.com/lumavpn/luma/log"

	"golang.zx2c4.com/wireguard/tun"
	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

const (
	// Queue length for outbound packet, arriving for read
	defaultOutQueueLen = 1 << 10
)

type DarwinEndpoint struct {
	*channel.Endpoint

	nt *tun.NativeTun

	mtu    uint32
	name   string
	offset int

	rSizes []int
	rBuffs [][]byte
	wBuffs [][]byte
	rMutex sync.Mutex
	wMutex sync.Mutex

	// rw is the io.ReadWriter for reading and writing packets.
	rw io.ReadWriter
	// once is used to perform the init action once when attaching
	once sync.Once
	// wg keeps track of running goroutines
	wg sync.WaitGroup
}

var _ GVisorTun = (*NativeTun)(nil)

func (t *NativeTun) NewEndpoint() (stack.LinkEndpoint, error) {
	ep := &DarwinEndpoint{
		mtu:    t.mtu,
		name:   t.name,
		offset: offset,
		rw:     t,
		rSizes: make([]int, 1),
		rBuffs: make([][]byte, 1),
		wBuffs: make([][]byte, 1),
	}
	forcedMTU := defaultMTU
	if t.mtu > 0 {
		forcedMTU = int(t.mtu)
	}
	nt, err := createTUN(t.name, forcedMTU)
	if err != nil {
		return nil, fmt.Errorf("create tun: %w", err)
	}
	ep.nt = nt.(*tun.NativeTun)
	ep.Endpoint = channel.New(defaultOutQueueLen, t.mtu, "")
	return ep, nil
}

var _ stack.LinkEndpoint = (*DarwinEndpoint)(nil)

func (e *DarwinEndpoint) Attach(dispatcher stack.NetworkDispatcher) {
	e.Endpoint.Attach(dispatcher)
	e.once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		e.wg.Add(2)
		go func() {
			e.outboundLoop(ctx)
			e.wg.Done()
		}()
		go func() {
			e.dispatchLoop(cancel)
			e.wg.Done()
		}()
	})
}

// dispatchLoop dispatches packets to upper layer.
func (e *DarwinEndpoint) dispatchLoop(cancel context.CancelFunc) {
	defer cancel()

	offset, mtu := e.offset, int(e.mtu)

	for {
		data := make([]byte, offset+mtu)

		n, err := e.rw.Read(data)
		if err != nil {
			break
		}

		if n == 0 || n > mtu {
			continue
		}

		if !e.IsAttached() {
			continue /* unattached, drop packet */
		}

		pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{
			Payload: buffer.MakeWithData(data[offset : offset+n]),
		})

		switch header.IPVersion(data[offset:]) {
		case header.IPv4Version:
			e.InjectInbound(header.IPv4ProtocolNumber, pkt)
		case header.IPv6Version:
			e.InjectInbound(header.IPv6ProtocolNumber, pkt)
		}
		pkt.DecRef()
	}
}

func (e *DarwinEndpoint) outboundLoop(ctx context.Context) {
	for {
		pkt := e.ReadContext(ctx)
		if pkt == nil {
			break
		}
		e.writePacket(pkt)
	}
}

// writePacket writes outbound packets to the io.Writer.
func (e *DarwinEndpoint) writePacket(pkt *stack.PacketBuffer) tcpip.Error {
	defer pkt.DecRef()

	buf := pkt.ToBuffer()
	defer buf.Release()
	if e.offset != 0 {
		v := buffer.NewViewWithData(make([]byte, e.offset))
		_ = buf.Prepend(v)
	}

	if _, err := e.rw.Write(buf.Flatten()); err != nil {
		return &tcpip.ErrInvalidEndpointState{}
	}
	return nil
}

func (t *DarwinEndpoint) Read(packet []byte) (int, error) {
	t.rMutex.Lock()
	defer t.rMutex.Unlock()
	t.rBuffs[0] = packet
	_, err := t.nt.Read(t.rBuffs, t.rSizes, t.offset)
	return t.rSizes[0], err
}

func (t *DarwinEndpoint) Write(packet []byte) (int, error) {
	t.wMutex.Lock()
	defer t.wMutex.Unlock()
	t.wBuffs[0] = packet
	return t.nt.Write(t.wBuffs, t.offset)
}

func (e *DarwinEndpoint) Name() string {
	name, _ := e.nt.Name()
	return name
}

func (e *DarwinEndpoint) WriteVectorised(buffers []*pool.Buffer) error {
	log.Debug("Here???")
	return nil
}

func (e *DarwinEndpoint) Close() {
	defer e.Endpoint.Close()
	e.nt.Close()
}
