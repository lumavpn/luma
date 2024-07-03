package outbound

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"

	C "github.com/lumavpn/luma/common"
	N "github.com/lumavpn/luma/common/net"
	"github.com/lumavpn/luma/component/ca"
	"github.com/lumavpn/luma/component/proxydialer"
	"github.com/lumavpn/luma/dialer"
	M "github.com/lumavpn/luma/metadata"
	P "github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/transport/socks5"
)

type Socks5 struct {
	*Base
	option *Socks5Option

	// unix indicates if socks5 over UDS is enabled.
	unix           bool
	tls            bool
	skipCertVerify bool
	tlsConfig      *tls.Config
}

type Socks5Option struct {
	BasicOption
	Addr           string `proxy:"addr"`
	Name           string `proxy:"name"`
	Server         string `proxy:"server"`
	Port           int    `proxy:"port"`
	UserName       string `proxy:"username,omitempty"`
	Password       string `proxy:"password,omitempty"`
	TLS            bool   `proxy:"tls,omitempty"`
	UDP            bool   `proxy:"udp,omitempty"`
	SkipCertVerify bool   `proxy:"skip-cert-verify,omitempty"`
	Fingerprint    string `proxy:"fingerprint,omitempty"`
}

func NewSocks5(opts *Socks5Option) (*Socks5, error) {
	var tlsConfig *tls.Config
	if opts.TLS {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: opts.SkipCertVerify,
			ServerName:         opts.Server,
		}

		var err error
		tlsConfig, err = ca.GetSpecifiedFingerprintTLSConfig(tlsConfig, opts.Fingerprint)
		if err != nil {
			return nil, err
		}
	}
	addr := opts.Addr
	return &Socks5{
		Base: &Base{
			name:     opts.Name,
			addr:     addr,
			proto:    proto.Proto_Socks5,
			udp:      opts.UDP,
			tfo:      opts.TFO,
			mpTcp:    opts.MPTCP,
			iface:    opts.Interface,
			rmark:    opts.RoutingMark,
			username: opts.UserName,
			password: opts.Password,
			prefer:   C.NewDNSPrefer(opts.IPVersion),
		},
		tls:            opts.TLS,
		skipCertVerify: opts.SkipCertVerify,
		tlsConfig:      tlsConfig,
		option:         opts,
		unix:           len(addr) > 0 && addr[0] == '/',
	}, nil
}

// StreamConnContext implements C.ProxyAdapter
func (ss *Socks5) StreamConnContext(ctx context.Context, c net.Conn, metadata *M.Metadata) (net.Conn, error) {
	if ss.tls {
		cc := tls.Client(c, ss.tlsConfig)
		err := cc.HandshakeContext(ctx)
		c = cc
		if err != nil {
			return nil, fmt.Errorf("%s connect error: %w", ss.addr, err)
		}
	}

	var user *socks5.User
	if ss.username != "" {
		user = &socks5.User{
			Username: ss.username,
			Password: ss.password,
		}
	}
	if _, err := socks5.ClientHandshake(c, serializesSocksAddr(metadata), socks5.CmdConnect, user); err != nil {
		return nil, err
	}
	return c, nil
}

func (ss *Socks5) DialContext(ctx context.Context, metadata *M.Metadata, opts ...dialer.Option) (_ P.Conn, err error) {
	return ss.DialContextWithDialer(ctx, dialer.NewDialer(ss.Base.DialOptions(opts...)...), metadata)
}

func (ss *Socks5) DialContextWithDialer(ctx context.Context, dialer P.Dialer, metadata *M.Metadata) (_ P.Conn, err error) {
	if ss.option.DialerProxy != nil {
		dialer = proxydialer.New(ss.option.DialerProxy, dialer, true)
		if err != nil {
			return nil, err
		}
	}
	c, err := dialer.DialContext(ctx, "tcp", ss.addr)
	if err != nil {
		return nil, fmt.Errorf("%s connect error: %w", ss.addr, err)
	}
	N.TCPKeepAlive(c)

	defer func(c net.Conn) {
		safeConnClose(c, err)
	}(c)

	c, err = ss.StreamConnContext(ctx, c, metadata)
	if err != nil {
		return nil, err
	}

	return NewConn(c, ss), nil
}

// SupportWithDialer implements C.ProxyAdapter
func (ss *Socks5) SupportWithDialer() M.Network {
	return M.TCP
}

func (ss *Socks5) ListenPacketContext(ctx context.Context, metadata *M.Metadata, opts ...dialer.Option) (_ P.PacketConn, err error) {
	var cDialer P.Dialer = dialer.NewDialer(ss.Base.DialOptions(opts...)...)
	if ss.option.DialerProxy != nil {
		cDialer = proxydialer.New(ss.option.DialerProxy, cDialer, true)
	}

	c, err := cDialer.DialContext(ctx, "tcp", ss.addr)
	if err != nil {
		err = fmt.Errorf("%s connect error: %w", ss.addr, err)
		return
	}

	if ss.tls {
		cc := tls.Client(c, ss.tlsConfig)
		ctx, cancel := context.WithTimeout(context.Background(), C.DefaultTLSTimeout)
		defer cancel()
		err = cc.HandshakeContext(ctx)
		c = cc
	}

	defer func(c net.Conn) {
		safeConnClose(c, err)
	}(c)

	N.TCPKeepAlive(c)
	var user *socks5.User
	if ss.username != "" {
		user = &socks5.User{
			Username: ss.username,
			Password: ss.password,
		}
	}

	udpAssocateAddr := socks5.AddrFromStdAddrPort(netip.AddrPortFrom(netip.IPv4Unspecified(), 0))
	bindAddr, err := socks5.ClientHandshake(c, udpAssocateAddr, socks5.CmdUDPAssociate, user)
	if err != nil {
		err = fmt.Errorf("client hanshake error: %w", err)
		return
	}

	// Support unspecified UDP bind address.
	bindUDPAddr := bindAddr.UDPAddr()
	if bindUDPAddr == nil {
		err = errors.New("invalid UDP bind address")
		return
	} else if bindUDPAddr.IP.IsUnspecified() {
		serverAddr, err := resolveUDPAddr(ctx, "udp", ss.Addr())
		if err != nil {
			return nil, err
		}

		bindUDPAddr.IP = serverAddr.IP
	}

	pc, err := cDialer.ListenPacket(ctx, "udp", "", bindUDPAddr.AddrPort())
	if err != nil {
		return
	}

	go func() {
		io.Copy(io.Discard, c)
		c.Close()
		// A UDP association terminates when the TCP connection that the UDP
		// ASSOCIATE request arrived on terminates. RFC1928
		pc.Close()
	}()

	return newPacketConn(&socksPacketConn{PacketConn: pc, rAddr: bindUDPAddr, tcpConn: c}, ss), nil
}

type socksPacketConn struct {
	net.PacketConn

	rAddr   net.Addr
	tcpConn net.Conn
}

func (pc *socksPacketConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	packet, err := socks5.EncodeUDPPacket(socks5.ParseAddrToSocksAddr(addr), b)
	if err != nil {
		return
	}
	return pc.PacketConn.WriteTo(packet, pc.rAddr)
}

func (pc *socksPacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, _, err := pc.PacketConn.ReadFrom(b)
	if err != nil {
		return 0, nil, err
	}

	addr, payload, err := socks5.DecodeUDPPacket(b)
	if err != nil {
		return 0, nil, err
	}

	udpAddr := addr.UDPAddr()
	if udpAddr == nil {
		return 0, nil, fmt.Errorf("convert %s to UDPAddr is nil", addr)
	}

	// due to DecodeUDPPacket is mutable, record addr length
	copy(b, payload)
	return n - len(addr) - 3, udpAddr, nil
}

func (pc *socksPacketConn) Close() error {
	pc.tcpConn.Close()
	return pc.PacketConn.Close()
}
