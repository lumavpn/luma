package proxydialer

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"strings"

	N "github.com/lumavpn/luma/common/net"
	"github.com/lumavpn/luma/component/resolver"
	"github.com/lumavpn/luma/dialer"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	C "github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/tunnel/statistic"
)

type proxyDialer struct {
	proxy     C.ProxyAdapter
	dialer    C.Dialer
	statistic bool
}

type ProxyDialer interface {
	C.Dialer
	Proxy() proxy.ProxyAdapter
}

func New(proxy C.ProxyAdapter, dialer C.Dialer, statistic bool) ProxyDialer {
	return proxyDialer{proxy: proxy, dialer: dialer, statistic: statistic}
}

func (p proxyDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	currentMeta := &M.Metadata{Type: proto.Proto_Inner}
	if err := currentMeta.SetRemoteAddress(address); err != nil {
		return nil, err
	}
	if strings.Contains(network, "udp") { // using in wireguard outbound
		if !currentMeta.Resolved() {
			ip, err := resolver.ResolveIP(ctx, currentMeta.Host)
			if err != nil {
				return nil, errors.New("can't resolve ip")
			}
			currentMeta.DstIP = ip
		}
		pc, err := p.listenPacket(ctx, currentMeta)
		if err != nil {
			return nil, err
		}
		return N.NewBindPacketConn(pc, currentMeta.UDPAddr()), nil
	}
	var conn C.Conn
	var err error
	if d, ok := p.dialer.(dialer.Dialer); ok { // first using old function to let mux work
		conn, err = p.proxy.DialContextWithDialer(ctx, p.dialer, currentMeta)
	} else {
		conn, err = p.proxy.DialContext(ctx, currentMeta, dialer.WithOption(d.Opt))
	}
	if err != nil {
		return nil, err
	}
	if p.statistic {
		conn = statistic.NewTCPTracker(conn, statistic.DefaultManager, currentMeta, nil, 0, 0, false)
	}
	return conn, err
}

func (p proxyDialer) Proxy() proxy.ProxyAdapter {
	return p.proxy
}

func (p proxyDialer) ListenPacket(ctx context.Context, network, address string, rAddrPort netip.AddrPort) (net.PacketConn, error) {
	currentMeta := &M.Metadata{Type: proto.Proto_Inner, DstIP: rAddrPort.Addr(), DstPort: rAddrPort.Port()}
	return p.listenPacket(ctx, currentMeta)
}

func (p proxyDialer) listenPacket(ctx context.Context, currentMeta *M.Metadata) (C.PacketConn, error) {
	var pc C.PacketConn
	var err error
	currentMeta.Network = M.UDP
	if d, ok := p.dialer.(dialer.Dialer); ok { // first using old function to let mux work
		pc, err = p.proxy.ListenPacketContext(ctx, currentMeta, dialer.WithOption(d.Opt))
	} else {
		pc, err = p.proxy.ListenPacketWithDialer(ctx, p.dialer, currentMeta)
	}
	if err != nil {
		return nil, err
	}
	if p.statistic {
		pc = statistic.NewUDPTracker(pc, statistic.DefaultManager, currentMeta, nil, 0, 0, false)
	}
	return pc, nil
}
