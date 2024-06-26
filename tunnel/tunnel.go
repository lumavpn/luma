package tunnel

import (
	"net/netip"
	"runtime"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/common/atomic"
)

type tunnel struct {
	fakeIPRange netip.Prefix
	status      atomic.TypedValue[TunnelStatus]
	tcpQueue    chan adapter.TCPConn
	udpQueue    chan adapter.UDPConn
}

type Tunnel interface {
	adapter.TransportHandler
}

// New returns a new instance of Tunnel
func New() Tunnel {
	t := &tunnel{
		status:   atomic.NewTypedValue[TunnelStatus](Suspend),
		tcpQueue: make(chan adapter.TCPConn),
		udpQueue: make(chan adapter.UDPConn),
	}
	go t.process()
	return t
}

func (t *tunnel) HandleTCP(conn adapter.TCPConn) {
	t.TCPIn() <- conn
}

func (t *tunnel) HandleUDP(conn adapter.UDPConn) {
	t.UDPIn() <- conn
}

// TCPIn return fan-in TCP queue.
func (t *tunnel) TCPIn() chan<- adapter.TCPConn {
	return t.tcpQueue
}

// UDPIn return fan-in UDP queue.
func (t *tunnel) UDPIn() chan<- adapter.UDPConn {
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
