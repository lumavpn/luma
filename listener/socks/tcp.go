package socks

import (
	"io"
	"net"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/conn"
	"github.com/lumavpn/luma/ipfilter"
	authStore "github.com/lumavpn/luma/listener/auth"
	"github.com/lumavpn/luma/listener/base"
	"github.com/lumavpn/luma/proxy/inbound"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/transport/socks4"
	"github.com/lumavpn/luma/transport/socks5"
	"github.com/lumavpn/luma/util"
)

type Listener struct {
	*base.BaseListener
}

func New(addr string, tunnel adapter.TransportHandler, options ...inbound.Option) (*Listener, error) {
	base, err := base.New(base.BaseOptions{
		Addr:   addr,
		Tunnel: tunnel,
	})
	if err != nil {
		return nil, err
	}
	ss := &Listener{base}
	if err := ss.ListenTCP(); err != nil {
		return nil, err
	}
	go ss.start(tunnel, options...)
	return &Listener{base}, nil
}

func (ss *Listener) start(tunnel adapter.TransportHandler, options ...inbound.Option) {
	for {
		c, err := ss.Accept()
		if err != nil {
			if ss.Closed() {
				break
			}
			continue
		}
		go handleSocks(c, tunnel, options...)
	}
}

func handleSocks(c net.Conn, tunnel adapter.TransportHandler, options ...inbound.Option) {
	util.TCPKeepAlive(c)

	bufConn := conn.NewBufConn(c)
	head, err := bufConn.Peek(1)
	if err != nil {
		c.Close()
		return
	}
	switch head[0] {
	case socks4.Version:
		handleSocks4(bufConn, tunnel, options...)
	case socks5.Version:
		handleSocks5(bufConn, tunnel, options...)
	default:
		c.Close()
	}
}

func handleSocks4(conn net.Conn, tunnel adapter.TransportHandler, options ...inbound.Option) {
	authenticator := authStore.Authenticator()
	if ipfilter.SkipAuthRemoteAddr(conn.RemoteAddr()) {
		authenticator = nil
	}
	addr, _, user, err := socks4.ServerHandshake(conn, authenticator)
	if err != nil {
		conn.Close()
		return
	}
	options = append(options, inbound.WithInUser(user))
	tunnel.HandleTCP(adapter.NewTCPConn(inbound.NewSocket(socks5.ParseAddr(addr), conn, proto.Proto_SOCKS4, options...)))
}

func handleSocks5(conn net.Conn, tunnel adapter.TransportHandler, options ...inbound.Option) {
	authenticator := authStore.Authenticator()
	if ipfilter.SkipAuthRemoteAddr(conn.RemoteAddr()) {
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
	options = append(options, inbound.WithInUser(user))
	tunnel.HandleTCP(adapter.NewTCPConn(inbound.NewSocket(target, conn, proto.Proto_SOCKS5, options...)))
}
