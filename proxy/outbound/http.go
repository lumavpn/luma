package outbound

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"

	C "github.com/lumavpn/luma/common"
	N "github.com/lumavpn/luma/common/net"
	"github.com/lumavpn/luma/component/ca"
	"github.com/lumavpn/luma/component/proxydialer"
	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/log"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/proto"
)

type Http struct {
	*Base
	tlsConfig *tls.Config
	option    *HttpOption
}

type HttpOption struct {
	BasicOption
	Name           string            `proxy:"name"`
	Server         string            `proxy:"server"`
	Port           int               `proxy:"port"`
	UserName       string            `proxy:"username,omitempty"`
	Password       string            `proxy:"password,omitempty"`
	TLS            bool              `proxy:"tls,omitempty"`
	SNI            string            `proxy:"sni,omitempty"`
	SkipCertVerify bool              `proxy:"skip-cert-verify,omitempty"`
	Fingerprint    string            `proxy:"fingerprint,omitempty"`
	Headers        map[string]string `proxy:"headers,omitempty"`
}

// StreamConnContext implements C.ProxyAdapter
func (h *Http) StreamConnContext(ctx context.Context, c net.Conn, metadata *M.Metadata) (net.Conn, error) {
	if h.tlsConfig != nil {
		cc := tls.Client(c, h.tlsConfig)
		err := cc.HandshakeContext(ctx)
		c = cc
		if err != nil {
			return nil, fmt.Errorf("%s connect error: %w", h.addr, err)
		}
	}

	if err := h.shakeHand(metadata, c); err != nil {
		return nil, err
	}
	return c, nil
}

// DialContext implements C.ProxyAdapter
func (h *Http) DialContext(ctx context.Context, metadata *M.Metadata, opts ...dialer.Option) (_ proxy.Conn, err error) {
	return h.DialContextWithDialer(ctx, dialer.NewDialer(h.Base.DialOptions(opts...)...), metadata)
}

// DialContextWithDialer implements C.ProxyAdapter
func (h *Http) DialContextWithDialer(ctx context.Context, dialer proxy.Dialer, metadata *M.Metadata) (_ proxy.Conn, err error) {
	if h.option.DialerProxy != nil {
		dialer = proxydialer.New(h.option.DialerProxy, dialer, true)
	}
	c, err := dialer.DialContext(ctx, "tcp", h.addr)
	if err != nil {
		return nil, fmt.Errorf("%s connect error: %w", h.addr, err)
	}
	N.TCPKeepAlive(c)

	defer func(c net.Conn) {
		safeConnClose(c, err)
	}(c)

	c, err = h.StreamConnContext(ctx, c, metadata)
	if err != nil {
		return nil, err
	}

	return NewConn(c, h), nil
}

// SupportWithDialer implements C.ProxyAdapter
func (h *Http) SupportWithDialer() M.Network {
	return M.TCP
}

func (h *Http) shakeHand(metadata *M.Metadata, rw io.ReadWriter) error {
	addr := metadata.RemoteAddress()
	HeaderString := "CONNECT " + addr + " HTTP/1.1\r\n"
	tempHeaders := map[string]string{
		"Host":             addr,
		"User-Agent":       "Go-http-client/1.1",
		"Proxy-Connection": "Keep-Alive",
	}

	for key, value := range h.option.Headers {
		tempHeaders[key] = value
	}

	if h.username != "" && h.password != "" {
		auth := h.username + ":" + h.password
		tempHeaders["Proxy-Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
		log.Debugf("Proxy-Authorization header is %v", tempHeaders["Proxy-Authorization"])
	}

	for key, value := range tempHeaders {
		HeaderString += key + ": " + value + "\r\n"
	}

	HeaderString += "\r\n"

	_, err := rw.Write([]byte(HeaderString))

	if err != nil {
		return err
	}

	resp, err := http.ReadResponse(bufio.NewReader(rw), nil)

	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	if resp.StatusCode == http.StatusProxyAuthRequired {
		return errors.New("HTTP need auth")
	}

	if resp.StatusCode == http.StatusMethodNotAllowed {
		return errors.New("CONNECT method not allowed by proxy")
	}

	if resp.StatusCode >= http.StatusInternalServerError {
		return errors.New(resp.Status)
	}

	return fmt.Errorf("can not connect remote err code: %d", resp.StatusCode)
}

func NewHttp(option HttpOption) (*Http, error) {
	var tlsConfig *tls.Config
	if option.TLS {
		sni := option.Server
		if option.SNI != "" {
			sni = option.SNI
		}
		var err error
		tlsConfig, err = ca.GetSpecifiedFingerprintTLSConfig(&tls.Config{
			InsecureSkipVerify: option.SkipCertVerify,
			ServerName:         sni,
		}, option.Fingerprint)
		if err != nil {
			return nil, err
		}
	}

	return &Http{
		Base: &Base{
			name:     option.Name,
			addr:     net.JoinHostPort(option.Server, strconv.Itoa(option.Port)),
			proto:    proto.Proto_HTTP,
			tfo:      option.TFO,
			mpTcp:    option.MPTCP,
			iface:    option.Interface,
			rmark:    option.RoutingMark,
			username: option.UserName,
			password: option.Password,
			prefer:   C.NewDNSPrefer(option.IPVersion),
		},
		tlsConfig: tlsConfig,
		option:    &option,
	}, nil
}
