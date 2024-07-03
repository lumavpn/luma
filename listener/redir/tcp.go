package redir

import (
	"net"

	"github.com/lumavpn/luma/adapter"
	N "github.com/lumavpn/luma/common/net"
	"github.com/lumavpn/luma/proxy/inbound"
	"github.com/lumavpn/luma/proxy/proto"
)

type Listener struct {
	listener net.Listener
	addr     string
	closed   bool
}

// RawAddress implements C.Listener
func (l *Listener) RawAddress() string {
	return l.addr
}

// Address implements C.Listener
func (l *Listener) Address() string {
	return l.listener.Addr().String()
}

// Close implements C.Listener
func (l *Listener) Close() error {
	l.closed = true
	return l.listener.Close()
}

func New(addr string, tunnel adapter.TransportHandler, additions ...inbound.Addition) (*Listener, error) {
	if len(additions) == 0 {
		additions = []inbound.Addition{
			inbound.WithInName("DEFAULT-REDIR"),
			inbound.WithSpecialRules(""),
		}
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	rl := &Listener{
		listener: l,
		addr:     addr,
	}

	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				if rl.closed {
					break
				}
				continue
			}
			go handleRedir(c, tunnel, additions...)
		}
	}()

	return rl, nil
}

func handleRedir(conn net.Conn, tunnel adapter.TransportHandler, additions ...inbound.Addition) {
	target, err := parserPacket(conn)
	if err != nil {
		conn.Close()
		return
	}
	N.TCPKeepAlive(conn)
	tunnel.HandleTCPConn(inbound.NewSocket(target, conn, proto.Proto_Redir, additions...))
}
