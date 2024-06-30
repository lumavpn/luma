package proxy

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

	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/dns"
	"github.com/lumavpn/luma/log"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/adapter"
	"github.com/lumavpn/luma/proxy/proto"
)

type HTTP struct {
	*Base
	option    HttpOption
	tlsConfig *tls.Config
}

type HttpOption struct {
	BasicOption
	TLS            bool              `proxy:"tls,omitempty"`
	SNI            string            `proxy:"sni,omitempty"`
	SkipCertVerify bool              `proxy:"skip-cert-verify,omitempty"`
	Headers        map[string]string `proxy:"headers,omitempty"`
}

func NewHTTP(opts HttpOption) (*HTTP, error) {
	var tlsConfig *tls.Config
	if opts.TLS {

	}
	return &HTTP{
		Base: &Base{
			name:     opts.Name,
			addr:     opts.Addr,
			proto:    proto.Proto_HTTP,
			tfo:      opts.TFO,
			mpTcp:    opts.MPTCP,
			iface:    opts.Interface,
			rmark:    opts.RoutingMark,
			username: opts.Username,
			password: opts.Password,
			prefer:   dns.NewDNSPrefer(opts.IPVersion),
		},
		option:    opts,
		tlsConfig: tlsConfig,
	}, nil
}

func (h *HTTP) StreamConnContext(ctx context.Context, c net.Conn, m *M.Metadata) (net.Conn, error) {
	if h.tlsConfig != nil {
		cc := tls.Client(c, h.tlsConfig)
		err := cc.HandshakeContext(ctx)
		c = cc
		if err != nil {
			return nil, fmt.Errorf("%s connect error: %w", h.addr, err)
		}
	}

	if err := h.shakeHand(m, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (h *HTTP) DialContext(ctx context.Context, metadata *M.Metadata, opts ...dialer.Option) (_ adapter.Conn, err error) {
	return h.DialContextWithDialer(ctx, dialer.NewDialer(h.Base.DialOptions(opts...)...), metadata)
}

func (h *HTTP) DialContextWithDialer(ctx context.Context, dialer Dialer, metadata *M.Metadata) (_ adapter.Conn, err error) {
	c, err := dialer.DialContext(ctx, "tcp", h.addr)
	if err != nil {
		return nil, fmt.Errorf("%s connect error: %w", h.addr, err)
	}
	setKeepAlive(c)

	defer func(c net.Conn) {
		safeConnClose(c, err)
	}(c)

	c, err = h.StreamConnContext(ctx, c, metadata)
	if err != nil {
		return nil, err
	}

	return NewConn(c, h), nil
}

func (h *HTTP) shakeHand(m *M.Metadata, rw io.ReadWriter) error {
	addr := m.DestinationAddress()
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
