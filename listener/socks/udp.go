package socks

import (
	"net"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/common/sockopt"
	"github.com/lumavpn/luma/conn"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy/inbound"
	"github.com/lumavpn/luma/proxy/protos"
	"github.com/lumavpn/luma/transport/socks5"
)

type UDPListener struct {
	packetConn net.PacketConn
	addr       string
	closed     bool
}

// NewUDP creates a new instance of UDPListener
func NewUDP(addr string, tunnel adapter.TransportHandler) (*UDPListener, error) {
	l, err := net.ListenPacket("udp", addr)
	if err != nil {
		return nil, err
	}

	if err := sockopt.UDPReuseaddr(l.(*net.UDPConn)); err != nil {
		log.Warnf("Failed to Reuse UDP Address: %s", err)
	}

	sl := &UDPListener{
		packetConn: l,
		addr:       addr,
	}
	conn := conn.NewEnhancePacketConn(l)
	go func() {
		for {
			data, put, remoteAddr, err := conn.WaitReadFrom()
			if err != nil {
				if put != nil {
					put()
				}
				if sl.closed {
					break
				}
				continue
			}
			handleSocksUDP(l, tunnel, data, put, remoteAddr)
		}
	}()

	return sl, nil
}

// RawAddress returns the raw address of the UDPListener
func (l *UDPListener) RawAddress() string {
	return l.addr
}

// RawAddress returns the address of the UDPListener
func (l *UDPListener) Address() string {
	return l.packetConn.LocalAddr().String()
}

// RawAddress closes the net.PacketConn underlying the UDPListener
func (l *UDPListener) Close() error {
	l.closed = true
	return l.packetConn.Close()
}

func handleSocksUDP(pc net.PacketConn, tunnel adapter.TransportHandler, buf []byte, put func(),
	addr net.Addr, options ...inbound.Option) {
	target, payload, err := socks5.DecodeUDPPacket(buf)
	if err != nil {
		if put != nil {
			put()
		}
		return
	}
	packet := &packet{
		pc:      pc,
		rAddr:   addr,
		payload: payload,
		put:     put,
	}
	tunnel.HandleUDPPacket(inbound.NewPacket(target, packet, protos.Protocol_SOCKS5, options...))
}
