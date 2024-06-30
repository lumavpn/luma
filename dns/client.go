package dns

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/dns/resolver"
	"github.com/lumavpn/luma/log"
	D "github.com/miekg/dns"
	"github.com/zhangyunhao116/fastrand"
)

type client struct {
	*D.Client
	addr      string
	host      string
	port      string
	iface     string
	proxyName string

	r *Resolver
}

type dnsClient interface {
	ExchangeContext(ctx context.Context, m *D.Msg) (msg *D.Msg, err error)
	Address() string
}

var _ dnsClient = (*client)(nil)

func (c *client) Address() string {
	if len(c.addr) != 0 {
		return c.addr
	}
	schema := "udp"
	if strings.HasPrefix(c.Client.Net, "tcp") {
		schema = "tcp"
		if strings.HasSuffix(c.Client.Net, "tls") {
			schema = "tls"
		}
	}

	c.addr = fmt.Sprintf("%s://%s", schema, net.JoinHostPort(c.host, c.port))
	return c.addr
}

func (c *client) ExchangeContext(ctx context.Context, m *D.Msg) (*D.Msg, error) {
	var ip netip.Addr
	var err error
	if c.r == nil {
		if ip, err = netip.ParseAddr(c.host); err != nil {
			return nil, fmt.Errorf("dns %s not a valid ip", c.host)
		}
	} else {
		ips, err := resolver.LookupIPWithResolver(ctx, c.host, c.r)
		if err != nil {
			return nil, fmt.Errorf("use default dns resolve failed: %w", err)
		} else if len(ips) == 0 {
			return nil, fmt.Errorf("%w: %s", resolver.ErrIPNotFound, c.host)
		}
		ip = ips[fastrand.Intn(len(ips))]
	}

	network := "udp"
	if strings.HasPrefix(c.Client.Net, "tcp") {
		network = "tcp"
	}

	var options []dialer.Option
	if c.iface != "" {
		options = append(options, dialer.WithInterface(c.iface))
	}

	dialHandler := getDialer(c.r, c.proxyName, options...)
	addr := net.JoinHostPort(ip.String(), c.port)
	conn, err := dialHandler(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = conn.Close()
	}()

	type result struct {
		msg *D.Msg
		err error
	}
	ch := make(chan result, 1)
	go func() {

		dConn := &D.Conn{
			Conn:         conn,
			UDPSize:      c.Client.UDPSize,
			TsigSecret:   c.Client.TsigSecret,
			TsigProvider: c.Client.TsigProvider,
		}

		msg, _, err := c.Client.ExchangeWithConn(m, dConn)

		if msg != nil && msg.Truncated && c.Client.Net == "" {
			tcpClient := *c.Client
			tcpClient.Net = "tcp"
			network = "tcp"
			log.Debugf("[DNS] Truncated reply from %s:%s for %s over UDP, retrying over TCP", c.host, c.port,
				m.Question[0].String())
			dConn.Conn, err = dialHandler(ctx, network, addr)
			if err != nil {
				ch <- result{msg, err}
				return
			}
			defer func() {
				_ = conn.Close()
			}()
			msg, _, err = tcpClient.ExchangeWithConn(m, dConn)
		}

		ch <- result{msg, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case ret := <-ch:
		return ret.msg, ret.err
	}
}
