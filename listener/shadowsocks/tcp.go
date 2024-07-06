package shadowsocks

import (
	"net"
	"strings"

	"github.com/lumavpn/luma/adapter"
	N "github.com/lumavpn/luma/common/net"
	LC "github.com/lumavpn/luma/listener/config"
	"github.com/lumavpn/luma/proxy/inbound"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/transport/shadowsocks/core"
	"github.com/lumavpn/luma/transport/socks5"
)

type Listener struct {
	closed       bool
	config       LC.ShadowsocksServer
	listeners    []net.Listener
	udpListeners []*UDPListener
	pickCipher   core.Cipher
}

var _listener *Listener

func New(config LC.ShadowsocksServer, tunnel adapter.TransportHandler) (*Listener, error) {
	pickCipher, err := core.PickCipher(config.Cipher, nil, config.Password)
	if err != nil {
		return nil, err
	}

	sl := &Listener{false, config, nil, nil, pickCipher}
	_listener = sl

	for _, addr := range strings.Split(config.Listen, ",") {
		addr := addr

		if config.Udp {
			//UDP
			ul, err := NewUDP(addr, pickCipher, tunnel)
			if err != nil {
				return nil, err
			}
			sl.udpListeners = append(sl.udpListeners, ul)
		}

		//TCP
		l, err := inbound.Listen("tcp", addr)
		if err != nil {
			return nil, err
		}
		sl.listeners = append(sl.listeners, l)

		go func() {
			for {
				c, err := l.Accept()
				if err != nil {
					if sl.closed {
						break
					}
					continue
				}
				N.TCPKeepAlive(c)
				go sl.HandleConn(c, tunnel)
			}
		}()
	}

	return sl, nil
}

func (l *Listener) Close() error {
	var retErr error
	for _, lis := range l.listeners {
		err := lis.Close()
		if err != nil {
			retErr = err
		}
	}
	for _, lis := range l.udpListeners {
		err := lis.Close()
		if err != nil {
			retErr = err
		}
	}
	return retErr
}

func (l *Listener) Config() string {
	return l.config.String()
}

func (l *Listener) AddrList() (addrList []net.Addr) {
	for _, lis := range l.listeners {
		addrList = append(addrList, lis.Addr())
	}
	for _, lis := range l.udpListeners {
		addrList = append(addrList, lis.LocalAddr())
	}
	return
}

func (l *Listener) HandleConn(conn net.Conn, tunnel adapter.TransportHandler, additions ...inbound.Addition) {
	conn = l.pickCipher.StreamConn(conn)
	conn = N.NewDeadlineConn(conn) // embed ss can't handle readDeadline correctly

	target, err := socks5.ReadAddr0(conn)
	if err != nil {
		_ = conn.Close()
		return
	}
	tunnel.HandleTCPConn(inbound.NewSocket(target, conn, proto.Protocol_SHADOWSOCKS, additions...))
}

func HandleShadowSocks(conn net.Conn, tunnel adapter.TransportHandler, additions ...inbound.Addition) bool {
	if _listener != nil && _listener.pickCipher != nil {
		go _listener.HandleConn(conn, tunnel, additions...)
		return true
	}
	return false
}