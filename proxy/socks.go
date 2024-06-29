package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"

	"github.com/lumavpn/luma/dialer"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/transport/socks5"
)

type Socks5 struct {
	*Base
	option *Socks5Option
	unix   bool
}

type Socks5Option struct {
	*BaseOption
}

// NewSocks5 creates a new Socks5-based proxy connector
func NewSocks5(opts *Socks5Option) (*Socks5, error) {
	addr := opts.Addr
	return &Socks5{
		Base: NewBase(opts.BaseOption),
		unix: len(addr) > 0 && addr[0] == '/',
	}, nil
}

func (ss *Socks5) DialContext(ctx context.Context, metadata *M.Metadata, opts ...dialer.Option) (c Conn, err error) {
	return ss.DialContextWithDialer(ctx, dialer.NewDialer(ss.Base.DialOptions(opts...)...), metadata)
}

// DialContextWithDialer implements C.ProxyAdapter
func (ss *Socks5) DialContextWithDialer(ctx context.Context, dialer Dialer, metadata *M.Metadata) (_ Conn, err error) {
	network := "tcp"
	if ss.unix {
		network = "unix"
	}

	c, err := dialer.DialContext(ctx, network, ss.Addr())
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", ss.Addr(), err)
	}
	setKeepAlive(c)
	defer func(c net.Conn) {
		safeConnClose(c, err)
	}(c)

	c, err = ss.StreamContext(ctx, c, metadata)
	if err != nil {
		return nil, err
	}
	return NewConn(c, ss), nil
}

func (ss *Socks5) StreamContext(ctx context.Context, c net.Conn, metadata *M.Metadata) (_ Conn, err error) {
	var user *socks5.User
	if ss.username != "" {
		user = &socks5.User{
			Username: ss.username,
			Password: ss.password,
		}
	}

	_, err = socks5.ClientHandshake(c, serializeSocksAddr(metadata), socks5.CmdConnect, user)
	return
}

func (ss *Socks5) ListenPacketContext(ctx context.Context, metadata *M.Metadata, opts ...dialer.Option) (_ PacketConn, err error) {
	var proxyDialer Dialer = dialer.NewDialer(ss.Base.DialOptions(opts...)...)
	ctx, cancel := context.WithTimeout(context.Background(), tcpConnectTimeout)
	defer cancel()

	c, err := dialer.DialContext(ctx, "tcp", ss.Addr())
	if err != nil {
		err = fmt.Errorf("connect to %s: %w", ss.Addr(), err)
		return
	}

	defer func(c net.Conn) {
		safeConnClose(c, err)
	}(c)

	setKeepAlive(c)
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

	pc, err := proxyDialer.ListenPacket(ctx, "udp", "", bindUDPAddr.AddrPort())
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