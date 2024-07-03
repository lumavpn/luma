package tunnel

import (
	"net"
	"net/netip"
	"runtime"
	"sync"

	"github.com/lumavpn/luma/adapter"
	C "github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/common/atomic"
	"github.com/lumavpn/luma/log"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxydialer"
	"github.com/lumavpn/luma/tunnel/nat"
	"github.com/lumavpn/luma/tunnel/sniffer"
)

type tunnel struct {
	fakeIPRange    netip.Prefix
	mode           C.TunnelMode
	natTable       nat.NatTable
	proxyDialer    proxydialer.ProxyDialer
	sniffingEnable bool
	status         atomic.TypedValue[TunnelStatus]
	tcpQueue       chan adapter.TCPConn
	udpQueue       chan adapter.PacketAdapter
	configMu       sync.Mutex
}

type Tunnel interface {
	adapter.TransportHandler
	FakeIPRange() netip.Prefix
	Mode() C.TunnelMode
	OnSuspend()
	OnInnerLoading()
	OnRunning()
	SetFakeIPRange(p netip.Prefix)
	SetMode(C.TunnelMode)
	UpdateSniffer(dispatcher *sniffer.SnifferDispatcher)
}

// New returns a new instance of Tunnel
func New(proxyDialer proxydialer.ProxyDialer) Tunnel {
	t := &tunnel{
		natTable:    nat.New(),
		proxyDialer: proxyDialer,
		status:      atomic.NewTypedValue[TunnelStatus](Suspend),
		tcpQueue:    make(chan adapter.TCPConn),
		udpQueue:    make(chan adapter.PacketAdapter, 200),
	}
	go t.process()
	return t
}

// Mode return current mode
func (t *tunnel) Mode() C.TunnelMode {
	return t.mode
}

// SetMode change the mode of tunnel
func (t *tunnel) SetMode(m C.TunnelMode) {
	log.Debugf("Setting tunnel mode to %s", m)
	t.mode = m
}

func (t *tunnel) FakeIPRange() netip.Prefix {
	return t.fakeIPRange
}

func (t *tunnel) SetFakeIPRange(fakeIPRange netip.Prefix) {
	t.fakeIPRange = fakeIPRange
}

func (t *tunnel) OnSuspend() {
	t.status.Store(Suspend)
}

func (t *tunnel) OnInnerLoading() {
	t.status.Store(Inner)
}

func (t *tunnel) OnRunning() {
	t.status.Store(Running)
}

func (t *tunnel) Status() TunnelStatus {
	return t.status.Load()
}

func (t *tunnel) UpdateSniffer(dispatcher *sniffer.SnifferDispatcher) {
	t.configMu.Lock()
	sniffer.Dispatcher = dispatcher
	t.sniffingEnable = dispatcher.Enable()
	t.configMu.Unlock()
}

func (t *tunnel) HandleTCPConn(c net.Conn, m *M.Metadata) {
	log.Debugf("Handling new TCP conn %s", m.JSON())
	t.handleTCPConn(adapter.NewTCPConn(c, m))
}

func (t *tunnel) HandleUDPPacket(packet adapter.UDPPacket, metadata *M.Metadata) {
	packetAdapter := adapter.NewPacketAdapter(packet, metadata)
	select {
	case t.udpQueue <- packetAdapter:
	default:
	}
}

// TCPIn return fan-in TCP queue.
func (t *tunnel) TCPIn() chan<- adapter.TCPConn {
	return t.tcpQueue
}

// UDPIn return fan-in UDP queue.
func (t *tunnel) UDPIn() chan<- adapter.PacketAdapter {
	return t.udpQueue
}

// processUDP starts a loop to handle UDP packets
func (t *tunnel) processUDP() {
	queue := t.udpQueue
	for conn := range queue {
		t.handleUDPConn(conn)
	}
}

func (t *tunnel) process() {
	numUDPWorkers := 4
	if num := runtime.GOMAXPROCS(0); num > numUDPWorkers {
		numUDPWorkers = num
	}
	for i := 0; i < numUDPWorkers; i++ {
		go t.processUDP()
	}

	queue := t.tcpQueue
	for conn := range queue {
		go t.handleTCPConn(conn)
	}
}
