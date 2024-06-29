package socks

import (
	"net"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/common/sockopt"
	"github.com/lumavpn/luma/conn"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy/inbound"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/transport/socks5"
)

type UDPListener struct {
	packetConn net.PacketConn
	addr       string
	closed     bool
}

// RawAddress implements C.Listener
func (l *UDPListener) RawAddress() string {
	return l.addr
}

// Address implements C.Listener
func (l *UDPListener) Address() string {
	return l.packetConn.LocalAddr().String()
}

// Close implements C.Listener
func (l *UDPListener) Close() error {
	l.closed = true
	return l.packetConn.Close()
}

func NewUDP(addr string, tunnel adapter.TransportHandler, options ...inbound.Option) (*UDPListener, error) {
	if len(options) == 0 {
		options = []inbound.Option{
			inbound.WithInName("DEFAULT-SOCKS"),
			inbound.WithSpecialRules(""),
		}
	}
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
			handleSocksUDP(l, tunnel, data, put, remoteAddr, options...)
		}
	}()

	return sl, nil
}

func handleSocksUDP(pc net.PacketConn, tunnel adapter.TransportHandler, buf []byte, put func(), addr net.Addr,
	options ...inbound.Option) {
	target, payload, err := socks5.DecodeUDPPacket(buf)
	if err != nil {
		// Unresolved UDP packet, return buffer to the pool
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
	tunnel.HandleUDP(adapter.NewPacketAdapter(inbound.NewPacket(target, packet, proto.Proto_SOCKS5, options...)))
}
