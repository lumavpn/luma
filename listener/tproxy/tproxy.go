package tproxy

import (
	"net"

	N "github.com/lumavpn/luma/common/net"
	"github.com/lumavpn/luma/proxy/inbound"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/transport/socks5"
	"github.com/lumavpn/luma/tunnel"
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

func (l *Listener) handleTProxy(conn net.Conn, tunnel tunnel.Tunnel, additions ...inbound.Addition) {
	target := socks5.ParseAddrToSocksAddr(conn.LocalAddr())
	N.TCPKeepAlive(conn)
	// TProxy's conn.LocalAddr() is target address, so we set from l.listener
	additions = append([]inbound.Addition{inbound.WithInAddr(l.listener.Addr())}, additions...)
	tunnel.HandleTCPConn(inbound.NewSocket(target, conn, proto.Proto_TProxy, additions...))
}

func New(addr string, tunnel tunnel.Tunnel, additions ...inbound.Addition) (*Listener, error) {
	if len(additions) == 0 {
		additions = []inbound.Addition{
			inbound.WithInName("DEFAULT-TPROXY"),
			inbound.WithSpecialRules(""),
		}
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	tl := l.(*net.TCPListener)
	rc, err := tl.SyscallConn()
	if err != nil {
		return nil, err
	}

	err = setsockopt(rc, addr)
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
			go rl.handleTProxy(c, tunnel, additions...)
		}
	}()

	return rl, nil
}
