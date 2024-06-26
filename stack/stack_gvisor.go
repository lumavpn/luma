//go:build with_gvisor

package stack

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/stack/tun"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
)

const defaultNIC tcpip.NICID = 1

type gVisor struct {
	options *Options
	tun     tun.Device
	stack   *stack.Stack
}

func NewGVisor(
	options *Options,
) (Stack, error) {
	log.Debug("Creating new gVisor stack")
	return &gVisor{
		options: options,
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
	linkEndpoint, err := tun.New(&tun.Options{
		//MTU: t.options.MTU,
	})
	if err != nil {
		return err
	}
	ipStack, err := newGVisorStack(linkEndpoint)
	if err != nil {
		return err
	}

	tcpForwarder := tcp.NewForwarder(ipStack, 0, 1024, func(r *tcp.ForwarderRequest) {

	})

	ipStack.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpForwarder.HandlePacket)

	udpForwarder := udp.NewForwarder(ipStack, func(request *udp.ForwarderRequest) {

	})

	ipStack.SetTransportProtocolHandler(udp.ProtocolNumber, udpForwarder.HandlePacket)

	return nil
}

func (t *gVisor) Stop() error {
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
