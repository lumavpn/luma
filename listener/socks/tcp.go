package socks

import (
	"io"
	"net"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/conn"
	"github.com/lumavpn/luma/ipfilter"
	"github.com/lumavpn/luma/listener/auth"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy/inbound"
	"github.com/lumavpn/luma/proxy/protos"
	"github.com/lumavpn/luma/transport/socks4"
	"github.com/lumavpn/luma/transport/socks5"
	"github.com/lumavpn/luma/util"
)

// Listener is a SOCKS inbound listener
type Listener struct {
	listener net.Listener
	options  []inbound.Option
	addr     string
	closed   bool
}

// New creates a new instance of Listener with the given inbound Options
func New(addr string, tunnel adapter.TransportHandler, options ...inbound.Option) (*Listener, error) {
	l, err := inbound.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	ss := &Listener{
		addr:     addr,
		listener: l,
		options:  options,
	}

	go ss.start(tunnel)
	return ss, nil
}

func (l *Listener) RawAddress() string {
	return l.addr
}

func (l *Listener) Address() string {
	return l.listener.Addr().String()
}

func (ss *Listener) start(tunnel adapter.TransportHandler, options ...inbound.Option) {
	isDefault := false
	if len(options) == 0 {
		isDefault = true
		options = []inbound.Option{
			inbound.WithInName("default-socks"),
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
		if isDefault {
			if !ipfilter.IsRemoteAddrDisAllowed(c.RemoteAddr()) {
				_ = c.Close()
				continue
			}
		}
		go handleSocks(c, tunnel, options...)
	}
}

func (ss *Listener) Close() error {
	ss.closed = true
	return ss.listener.Close()
}

func handleSocks(c net.Conn, tunnel adapter.TransportHandler, options ...inbound.Option) {
	log.Debug("handleSocks called")
	util.TCPKeepAlive(c)
	bufConn := conn.NewBuffConn(c)
	head, err := bufConn.Peek(1)
	if err != nil {
		c.Close()
		return
	}

	switch head[0] {
	case socks4.Version:
		HandleSocks4(bufConn, tunnel, options...)
	case socks5.Version:
		HandleSocks5(bufConn, tunnel, options...)
	default:
		c.Close()
	}
}

func HandleSocks4(conn net.Conn, tunnel adapter.TransportHandler, options ...inbound.Option) {
	authenticator := auth.Authenticator()
	addr, _, _, err := socks4.ServerHandshake(conn, authenticator)
	if err != nil {
		conn.Close()
		return
	}
	tunnel.HandleTCPConn(inbound.NewSocket(socks5.ParseAddr(addr), conn, protos.Protocol_SOCKS4))
}

func HandleSocks5(conn net.Conn, tunnel adapter.TransportHandler, options ...inbound.Option) {
	authenticator := auth.Authenticator()
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
	tunnel.HandleTCPConn(inbound.NewSocket(target, conn, protos.Protocol_SOCKS5, options...))
}
