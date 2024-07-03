package socks

import (
	"io"
	"net"
	"sync"

	"github.com/lumavpn/luma/adapter"
	N "github.com/lumavpn/luma/common/net"
	authStore "github.com/lumavpn/luma/listener/auth"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy/inbound"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/transport/socks4"
	"github.com/lumavpn/luma/transport/socks5"
)

type Listener struct {
	listener net.Listener
	addr     string
	closed   bool
	mu       sync.Mutex
}

func New(addr string, tunnel adapter.TransportHandler, additions ...inbound.Addition) (*Listener, error) {
	l, err := inbound.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	ss := &Listener{
		addr:     addr,
		listener: l,
	}

	go ss.start(tunnel, additions...)
	return ss, nil
}

func (l *Listener) RawAddress() string {
	return l.addr
}

func (l *Listener) Address() string {
	return l.listener.Addr().String()
}

func (ss *Listener) start(tunnel adapter.TransportHandler, additions ...inbound.Addition) {
	isDefault := false
	if len(additions) == 0 {
		isDefault = true
		additions = []inbound.Addition{
			inbound.WithInName("DEFAULT-SOCKS"),
			inbound.WithSpecialRules(""),
		}
	}
	for {
		c, err := ss.listener.Accept()
		if err != nil {
			if ss.closed {
				break
			}
			continue
		}
		if isDefault { // only apply on default listener
			if !inbound.IsRemoteAddrDisAllowed(c.RemoteAddr()) {
				_ = c.Close()
				continue
			}
		}
		go handleSocks(c, tunnel, additions...)
	}
}

func (ss *Listener) Close() error {
	ss.closed = true
	return ss.listener.Close()
}

func handleSocks(conn net.Conn, tunnel adapter.TransportHandler, additions ...inbound.Addition) {
	log.Debug("handleSocks called")
	N.TCPKeepAlive(conn)
	bufConn := N.NewBufferedConn(conn)
	head, err := bufConn.Peek(1)
	if err != nil {
		conn.Close()
		return
	}

	switch head[0] {
	case socks4.Version:
		HandleSocks4(bufConn, tunnel, additions...)
	case socks5.Version:
		HandleSocks5(bufConn, tunnel, additions...)
	default:
		conn.Close()
	}
}

func HandleSocks4(conn net.Conn, tunnel adapter.TransportHandler, additions ...inbound.Addition) {
	authenticator := authStore.Authenticator()
	if inbound.SkipAuthRemoteAddr(conn.RemoteAddr()) {
		authenticator = nil
	}
	addr, _, user, err := socks4.ServerHandshake(conn, authenticator)
	if err != nil {
		conn.Close()
		return
	}
	additions = append(additions, inbound.WithInUser(user))
	tunnel.HandleTCPConn(inbound.NewSocket(socks5.ParseAddr(addr), conn, proto.Proto_Socks4, additions...))
}

func HandleSocks5(conn net.Conn, tunnel adapter.TransportHandler, additions ...inbound.Addition) {
	authenticator := authStore.Authenticator()
	if inbound.SkipAuthRemoteAddr(conn.RemoteAddr()) {
		authenticator = nil
	}
	target, command, user, err := socks5.ServerHandshake(conn, authenticator)
	if err != nil {
		conn.Close()
		return
	}
	if command == socks5.CmdUDPAssociate {
		defer conn.Close()
		io.Copy(io.Discard, conn)
		return
	}
	additions = append(additions, inbound.WithInUser(user))
	tunnel.HandleTCPConn(inbound.NewSocket(target, conn, proto.Proto_Socks5, additions...))
}
