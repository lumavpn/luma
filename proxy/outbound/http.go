package outbound

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
)

type HTTP struct {
	*Base
	tlsConfig *tls.Config
	username  string
	password  string
}

type HttpOptions struct {
	*BaseOptions
	Username string
	Password string
}

// NewHTTP creates a new HTTP-based outbound proxy.Proxy
func NewHTTP(opts HttpOptions) (*HTTP, error) {
	return &HTTP{
		Base: &Base{
			addr: opts.Addr,
			name: opts.Name,
		},
		username: opts.Username,
		password: opts.Password,
	}, nil
}

func (h *HTTP) StreamConnContext(ctx context.Context, c net.Conn, m *metadata.Metadata) (net.Conn, error) {
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

// DialContext connects to the address on the network specified by Metadata
func (h *HTTP) DialContext(ctx context.Context, m *metadata.Metadata, opts ...dialer.Option) (_ proxy.Conn, err error) {
	c, err := dialer.DialContext(ctx, "tcp", h.addr)
	if err != nil {
		return nil, fmt.Errorf("%s connect error: %w", h.addr, err)
	}
	setKeepAlive(c)

	defer func(c net.Conn) {
		safeConnClose(c, err)
	}(c)

	c, err = h.StreamConnContext(ctx, c, m)
	if err != nil {
		return nil, err
	}

	return NewConn(c, h), nil
}

func (h *HTTP) shakeHand(m *metadata.Metadata, rw io.ReadWriter) error {
	addr := m.DestinationAddress()
	req := &http.Request{
		Method: http.MethodConnect,
		URL: &url.URL{
			Host: addr,
		},
		Host: addr,
		Header: http.Header{
			"Proxy-Connection": []string{"Keep-Alive"},
		},
	}

	if h.username != "" && h.password != "" {
		req.Header.Set("Proxy-Authorization", fmt.Sprintf("Basic %s", basicAuth(h.username, h.password)))
	}

	if err := req.Write(rw); err != nil {
		return err
	}

	resp, err := http.ReadResponse(bufio.NewReader(rw), req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusProxyAuthRequired:
		return errors.New("HTTP auth required by proxy")
	case http.StatusMethodNotAllowed:
		return errors.New("CONNECT method not allowed by proxy")
	default:
		return fmt.Errorf("HTTP connect status: %s", resp.Status)
	}
}
