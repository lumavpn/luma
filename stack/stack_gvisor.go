//go:build with_gvisor

package stack

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/lumavpn/luma/log"
	"github.com/sagernet/sing/common/bufio"
	"github.com/sagernet/sing/common/canceler"
	M "github.com/sagernet/sing/common/metadata"
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

const defaultNIC tcpip.NICID = 1

type gVisor struct {
	ctx                    context.Context
	config                 *Config
	handler                Handler
	endpointIndependentNat bool
	tun                    GVisorTun
	broadcastAddr          netip.Addr
	udpTimeout             int64
	stack                  *stack.Stack
	endpoint               stack.LinkEndpoint
}

type GVisorTun interface {
	Tun
	NewEndpoint() (stack.LinkEndpoint, error)
}

func NewGVisor(
	cfg *Config,
) (Stack, error) {
	gTun, isGTun := cfg.Tun.(GVisorTun)
	if !isGTun {
		return nil, errors.New("gVisor stack is unsupported on current platform")
	}
	log.Debug("Creating new gVisor stack")
	return &gVisor{
		ctx:                    cfg.Context,
		config:                 cfg,
		endpointIndependentNat: cfg.EndpointIndependentNat,
		udpTimeout:             cfg.UDPTimeout,
		handler:                cfg.Handler,
		broadcastAddr:          BroadcastAddr(cfg.TunOptions.Inet4Address),
		tun:                    gTun,
	}, nil
}

func (t *gVisor) Start(ctx context.Context) error {
	linkEndpoint, err := t.tun.NewEndpoint()
	if err != nil {
		return err
	}
	linkEndpoint = &LinkEndpointFilter{linkEndpoint, t.broadcastAddr, t.tun}

	ipStack, err := newGVisorStack(linkEndpoint)
	if err != nil {
		return err
	}

	tcpForwarder := tcp.NewForwarder(ipStack, 0, 1024, func(r *tcp.ForwarderRequest) {
		var wq waiter.Queue
		handshakeCtx, cancel := context.WithCancel(context.Background())
		go func() {
			select {
			case <-t.ctx.Done():
				wq.Notify(wq.Events())
			case <-handshakeCtx.Done():
			}
		}()
		endpoint, err := r.CreateEndpoint(&wq)
		cancel()
		if err != nil {
			r.Complete(true)
			return
		}
		r.Complete(false)
		endpoint.SocketOptions().SetKeepAlive(true)
		keepAliveIdle := tcpip.KeepaliveIdleOption(15 * time.Second)
		endpoint.SetSockOpt(&keepAliveIdle)
		keepAliveInterval := tcpip.KeepaliveIntervalOption(15 * time.Second)
		endpoint.SetSockOpt(&keepAliveInterval)
		tcpConn := gonet.NewTCPConn(&wq, endpoint)
		lAddr := tcpConn.RemoteAddr()
		rAddr := tcpConn.LocalAddr()
		if lAddr == nil || rAddr == nil {
			tcpConn.Close()
			return
		}
		go func() {
			var metadata M.Metadata
			metadata.Source = M.SocksaddrFromNet(lAddr)
			metadata.Destination = M.SocksaddrFromNet(rAddr)
			hErr := t.handler.NewConnection(t.ctx, &gTCPConn{tcpConn}, metadata)
			if hErr != nil {
				endpoint.Abort()
			}
		}()
	})
	ipStack.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpForwarder.HandlePacket)
	if !t.endpointIndependentNat {
		udpForwarder := udp.NewForwarder(ipStack, func(request *udp.ForwarderRequest) {
			var wq waiter.Queue
			endpoint, err := request.CreateEndpoint(&wq)
			if err != nil {
				return
			}
			udpConn := gonet.NewUDPConn(&wq, endpoint)
			lAddr := udpConn.RemoteAddr()
			rAddr := udpConn.LocalAddr()
			if lAddr == nil || rAddr == nil {
				endpoint.Abort()
				return
			}
			gConn := &gUDPConn{UDPConn: udpConn}
			go func() {
				var metadata M.Metadata
				metadata.Source = M.SocksaddrFromNet(lAddr)
				metadata.Destination = M.SocksaddrFromNet(rAddr)
				ctx, conn := canceler.NewPacketConn(t.ctx, bufio.NewUnbindPacketConnWithAddr(gConn, metadata.Destination), time.Duration(t.udpTimeout)*time.Second)
				hErr := t.handler.NewPacketConnection(ctx, conn, metadata)
				if hErr != nil {
					endpoint.Abort()
				}
			}()
		})
		ipStack.SetTransportProtocolHandler(udp.ProtocolNumber, udpForwarder.HandlePacket)
	} else {
		ipStack.SetTransportProtocolHandler(udp.ProtocolNumber, NewUDPForwarder(t.ctx, ipStack, t.handler, t.udpTimeout).HandlePacket)
	}

	t.stack = ipStack
	t.endpoint = linkEndpoint
	return nil
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

func (t *gVisor) Close() error {
	t.endpoint.Attach(nil)
	t.stack.Close()
	for _, endpoint := range t.stack.CleanupEndpoints() {
		endpoint.Abort()
	}
	return nil
}

func AddressFromAddr(destination netip.Addr) tcpip.Address {
	if destination.Is6() {
		return tcpip.AddrFrom16(destination.As16())
	} else {
		return tcpip.AddrFrom4(destination.As4())
	}
}

func AddrFromAddress(address tcpip.Address) netip.Addr {
	if address.Len() == 16 {
		return netip.AddrFrom16(address.As16())
	} else {
		return netip.AddrFrom4(address.As4())
	}
}
