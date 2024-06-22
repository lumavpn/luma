package outbound

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/protos"
	"github.com/lumavpn/luma/transport/socks5"
)

type Socks5 struct {
	*Base

	username string
	password string

	// unix indicates if socks5 over UDS is enabled.
	unix           bool
	tls            bool
	skipCertVerify bool
	tlsConfig      *tls.Config
}

type Socks5Options struct {
	BaseOptions
	Username       string `proxy:"username,omitempty"`
	Password       string `proxy:"password,omitempty"`
	Server         string `proxy:"server"`
	TLS            bool   `proxy:"tls,omitempty"`
	UDP            bool   `proxy:"udp,omitempty"`
	SkipCertVerify bool   `proxy:"skip-cert-verify,omitempty"`
	Fingerprint    string `proxy:"fingerprint,omitempty"`
}

func NewSocks5(opts Socks5Options) (*Socks5, error) {
	var tlsConfig *tls.Config
	if opts.TLS {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: opts.SkipCertVerify,
			ServerName:         opts.Server,
		}

	}
	addr := opts.Addr
	return &Socks5{
		Base: &Base{
			name:  opts.Name,
			addr:  opts.Addr,
			at:    protos.AdapterType_Socks5,
			proto: protos.Protocol_SOCKS5,
		},
		username:       opts.Username,
		password:       opts.Password,
		tls:            opts.TLS,
		skipCertVerify: opts.SkipCertVerify,
		tlsConfig:      tlsConfig,

		unix: len(addr) > 0 && addr[0] == '/',
	}, nil
}

func (ss *Socks5) StreamConnContext(ctx context.Context, c net.Conn, m *metadata.Metadata) (net.Conn, error) {
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
	if _, err := socks5.ClientHandshake(c, serializesSocksAddr(m), socks5.CmdConnect, user); err != nil {
		return nil, err
	}
	return c, nil
}

func (ss *Socks5) DialContext(ctx context.Context, m *metadata.Metadata, opts ...dialer.Option) (_ proxy.Conn, err error) {
	c, err := dialer.DialContext(ctx, "tcp", ss.addr)
	if err != nil {
		return nil, fmt.Errorf("%s connect error: %w", ss.addr, err)
	}
	setKeepAlive(c)

	defer func(c net.Conn) {
		safeConnClose(c, err)
	}(c)

	c, err = ss.StreamConnContext(ctx, c, m)
	if err != nil {
		return nil, err
	}

	return NewConn(c, ss), nil
}
