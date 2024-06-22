package tunnel

import (
	"net"
	"net/netip"
	"runtime"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/common/atomic"
	"github.com/lumavpn/luma/conn"
	"github.com/lumavpn/luma/log"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxydialer"
	"github.com/lumavpn/luma/tunnel/nat"
)

type tunnel struct {
	fakeIPRange netip.Prefix
	mode        common.TunnelMode
	proxyDialer proxydialer.ProxyDialer
	status      atomic.TypedValue[TunnelStatus]
	tcpQueue    chan adapter.TCPConn
	udpQueue    chan adapter.PacketAdapter
	natTable    nat.NatTable
}

type Tunnel interface {
	FakeIPRange() netip.Prefix
	SetFakeIPRange(p netip.Prefix)

	HandleTCPConn(c net.Conn, metadata *M.Metadata)
	HandleUDPPacket(packet adapter.UDPPacket, metadata *M.Metadata)

	SetMode(m common.TunnelMode)

	// SetStatus sets the current status of the Tunnel
	SetStatus(s TunnelStatus)
	// Status returns the current status of the Tunnel
	Status() TunnelStatus
}

// New returns a new instance of Tunnel
func New(proxyDialer proxydialer.ProxyDialer) Tunnel {
	t := &tunnel{
		natTable:    nat.New(),
		proxyDialer: proxyDialer,
		status:      atomic.NewTypedValue[TunnelStatus](Suspend),
		tcpQueue:    make(chan adapter.TCPConn),
		udpQueue:    make(chan adapter.PacketAdapter),
	}
	go t.process()
	return t
}

func (t *tunnel) HandleTCPConn(c net.Conn, metadata *M.Metadata) {
	connCtx, err := conn.NewConnContext(c, metadata)
	if err != nil {
		log.Error(err)
		return
	}
	t.handleTCPConn(connCtx)
}

func (t *tunnel) HandleUDPPacket(packet adapter.UDPPacket, metadata *M.Metadata) {
	packetAdapter := adapter.NewPacketAdapter(packet, metadata)
	select {
	case t.udpQueue <- packetAdapter:
	default:
	}
}

func (t *tunnel) SetFakeIPRange(p netip.Prefix) {
	t.fakeIPRange = p
}

func (t *tunnel) FakeIPRange() netip.Prefix {
	return t.fakeIPRange
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

// Mode return current mode
func (t *tunnel) Mode() common.TunnelMode {
	return t.mode
}

// SetMode change the mode of tunnel
func (t *tunnel) SetMode(m common.TunnelMode) {
	log.Debugf("Setting tunnel mode to %s", m)
	t.mode = m
}

// SetStatus sets the current status of the Tunnel
func (t *tunnel) SetStatus(s TunnelStatus) {
	t.status.Store(s)
}

// Status returns the current status of the Tunnel
func (t *tunnel) Status() TunnelStatus {
	return t.status.Load()
}
