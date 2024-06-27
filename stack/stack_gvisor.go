//go:build with_gvisor

package stack

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/stack/tun"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
)

const (
	defaultNIC           tcpip.NICID = 1
	defaultWndSize                   = 0
	maxConnAttempts                  = 2 << 10
	tcpKeepaliveCount                = 9
	tcpKeepaliveIdle                 = 60 * time.Second
	tcpKeepaliveInterval             = 30 * time.Second
)

type gVisor struct {
	handler  Handler
	options  *Options
	tun      tun.GVisorTun
	stack    *stack.Stack
	endpoint stack.LinkEndpoint
}

func NewGVisor(
	options *Options,
) (Stack, error) {
	gTun, isGTun := options.Tun.(tun.GVisorTun)
	if !isGTun {
		return nil, errors.New("gVisor stack is unsupported on current platform")
	}
	log.Debug("Creating new gVisor stack")
	return &gVisor{
		tun:      gTun,
		endpoint: options.Device,
		handler:  options.Handler,
		options:  options,
	}, nil
}

func newGVisorStack(ep stack.LinkEndpoint) (*stack.Stack, error) {
	ipStack := stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			ipv4.NewProtocol,
			ipv6.NewProtocol,
		},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
			udp.NewProtocol,
			icmp.NewProtocol4,
			icmp.NewProtocol6,
		},
	})
	tErr := ipStack.CreateNIC(defaultNIC, ep)
	if tErr != nil {
		return nil, fmt.Errorf("create nic: %v", wrapStackError(tErr))
	}
	ipStack.SetRouteTable([]tcpip.Route{
		{Destination: header.IPv4EmptySubnet, NIC: defaultNIC},
		{Destination: header.IPv6EmptySubnet, NIC: defaultNIC},
	})
	ipStack.SetSpoofing(defaultNIC, true)
	ipStack.SetPromiscuousMode(defaultNIC, true)
	bufSize := 20 * 1024
	ipStack.SetTransportProtocolOption(tcp.ProtocolNumber, &tcpip.TCPReceiveBufferSizeRangeOption{
		Min:     1,
		Default: bufSize,
		Max:     bufSize,
	})
	ipStack.SetTransportProtocolOption(tcp.ProtocolNumber, &tcpip.TCPSendBufferSizeRangeOption{
		Min:     1,
		Default: bufSize,
		Max:     bufSize,
	})
	sOpt := tcpip.TCPSACKEnabled(true)
	ipStack.SetTransportProtocolOption(tcp.ProtocolNumber, &sOpt)
	mOpt := tcpip.TCPModerateReceiveBufferOption(true)
	ipStack.SetTransportProtocolOption(tcp.ProtocolNumber, &mOpt)
	return ipStack, nil
}

func (t *gVisor) Start(ctx context.Context) error {
	linkEndpoint, err := t.tun.NewEndpoint()
	if err != nil {
		return err
	}

	ipStack, err := newGVisorStack(linkEndpoint)
	if err != nil {
		return err
	}

	tcpForwarder := tcp.NewForwarder(ipStack, 0, 1024, func(r *tcp.ForwarderRequest) {
		var (
			wq waiter.Queue
			id = r.ID()
		)
		endpoint, err := r.CreateEndpoint(&wq)
		if err != nil {
			r.Complete(true)
			return
		}
		r.Complete(false)

		err = setSocketOptions(ipStack, endpoint)

		conn := adapter.NewTCPConn(gonet.NewTCPConn(&wq, endpoint), id)

		// go t.Handler.NewConnection
		hErr := t.handler.NewConnection(ctx, conn)

		if hErr != nil {
			endpoint.Abort()
		}

	})

	ipStack.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpForwarder.HandlePacket)

	udpForwarder := udp.NewForwarder(ipStack, func(r *udp.ForwarderRequest) {
		var (
			wq waiter.Queue
			id = r.ID()
		)
		endpoint, err := r.CreateEndpoint(&wq)
		if err != nil {
			return
		}
		udpConn := gonet.NewUDPConn(&wq, endpoint)
		conn := adapter.NewUDPConn(udpConn, id)

		// go t.Handler.NewPacketConnection
		t.handler.NewPacketConnection(ctx, conn)
	})

	ipStack.SetTransportProtocolHandler(udp.ProtocolNumber, udpForwarder.HandlePacket)

	t.endpoint = linkEndpoint
	t.stack = ipStack

	return nil
}

func setSocketOptions(s *stack.Stack, ep tcpip.Endpoint) tcpip.Error {
	{ /* TCP keepalive options */
		ep.SocketOptions().SetKeepAlive(true)

		idle := tcpip.KeepaliveIdleOption(tcpKeepaliveIdle)
		if err := ep.SetSockOpt(&idle); err != nil {
			return err
		}

		interval := tcpip.KeepaliveIntervalOption(tcpKeepaliveInterval)
		if err := ep.SetSockOpt(&interval); err != nil {
			return err
		}

		if err := ep.SetSockOptInt(tcpip.KeepaliveCountOption, tcpKeepaliveCount); err != nil {
			return err
		}
	}
	{ /* TCP recv/send buffer size */
		var ss tcpip.TCPSendBufferSizeRangeOption
		if err := s.TransportProtocolOption(header.TCPProtocolNumber, &ss); err == nil {
			ep.SocketOptions().SetSendBufferSize(int64(ss.Default), false)
		}

		var rs tcpip.TCPReceiveBufferSizeRangeOption
		if err := s.TransportProtocolOption(header.TCPProtocolNumber, &rs); err == nil {
			ep.SocketOptions().SetReceiveBufferSize(int64(rs.Default), false)
		}
	}
	return nil
}

func (t *gVisor) Stop() error {
	log.Debug("Closing gvisor stack..")
	t.endpoint.Attach(nil)
	t.stack.Close()
	for _, endpoint := range t.stack.CleanupEndpoints() {
		endpoint.Abort()
	}
	return nil
}

func wrapStackError(err tcpip.Error) error {
	switch err.(type) {
	case *tcpip.ErrClosedForSend,
		*tcpip.ErrClosedForReceive,
		*tcpip.ErrAborted:
		return net.ErrClosed
	}
	return errors.New(err.String())
}
