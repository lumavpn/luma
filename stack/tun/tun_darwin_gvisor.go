//go:build !with_wireguard && with_gvisor && darwin

package tun

import (
	"github.com/lumavpn/luma/common/bufio"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

var _ GVisorTun = (*NativeTun)(nil)

func (t *NativeTun) NewEndpoint() (stack.LinkEndpoint, error) {
	return &DarwinEndpoint{tun: t}, nil
}

var _ stack.LinkEndpoint = (*DarwinEndpoint)(nil)

type DarwinEndpoint struct {
	tun        *NativeTun
	dispatcher stack.NetworkDispatcher
}

func (e *DarwinEndpoint) MTU() uint32 {
	return e.tun.mtu
}

// SetMTU update the maximum transmission unit for the endpoint.
func (e *DarwinEndpoint) SetMTU(mtu uint32) {
	e.tun.mtu = mtu
}

func (e *DarwinEndpoint) MaxHeaderLength() uint16 {
	return 0
}

func (e *DarwinEndpoint) LinkAddress() tcpip.LinkAddress {
	return ""
}

// Close is called when the endpoint is removed from a stack.
func (e *DarwinEndpoint) Close() {

}

func (e *DarwinEndpoint) SetLinkAddress(addr tcpip.LinkAddress) {

}

func (e *DarwinEndpoint) Capabilities() stack.LinkEndpointCapabilities {
	return stack.CapabilityRXChecksumOffload
}

func (e *DarwinEndpoint) Attach(dispatcher stack.NetworkDispatcher) {
	if dispatcher == nil && e.dispatcher != nil {
		e.dispatcher = nil
		return
	}
	if dispatcher != nil && e.dispatcher == nil {
		e.dispatcher = dispatcher
		go e.dispatchLoop()
	}
}

func (e *DarwinEndpoint) dispatchLoop() {
	packetBuffer := make([]byte, e.tun.mtu+PacketOffset)
	for {
		n, err := e.tun.tunFile.Read(packetBuffer)
		if err != nil {
			break
		}
		packet := packetBuffer[PacketOffset:n]
		var networkProtocol tcpip.NetworkProtocolNumber
		switch header.IPVersion(packet) {
		case header.IPv4Version:
			networkProtocol = header.IPv4ProtocolNumber
			if header.IPv4(packet).DestinationAddress().As4() == e.tun.inet4Address {
				e.tun.tunFile.Write(packetBuffer[:n])
				continue
			}
		case header.IPv6Version:
			networkProtocol = header.IPv6ProtocolNumber
			if header.IPv6(packet).DestinationAddress().As16() == e.tun.inet6Address {
				e.tun.tunFile.Write(packetBuffer[:n])
				continue
			}
		default:
			e.tun.tunFile.Write(packetBuffer[:n])
			continue
		}
		pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{
			Payload:           buffer.MakeWithData(packetBuffer[4:n]),
			IsForwardedPacket: true,
		})
		pkt.NetworkProtocolNumber = networkProtocol
		dispatcher := e.dispatcher
		if dispatcher == nil {
			pkt.DecRef()
			return
		}
		dispatcher.DeliverNetworkPacket(networkProtocol, pkt)
		pkt.DecRef()
	}
}

func (e *DarwinEndpoint) IsAttached() bool {
	return e.dispatcher != nil
}

func (e *DarwinEndpoint) Wait() {
}

func (e *DarwinEndpoint) ARPHardwareType() header.ARPHardwareType {
	return header.ARPHardwareNone
}

func (e *DarwinEndpoint) AddHeader(buffer *stack.PacketBuffer) {
}

func (e *DarwinEndpoint) ParseHeader(ptr *stack.PacketBuffer) bool {
	return true
}

func (e *DarwinEndpoint) WritePackets(packetBufferList stack.PacketBufferList) (int, tcpip.Error) {
	var n int
	for _, packet := range packetBufferList.AsSlice() {
		_, err := bufio.WriteVectorised(e.tun, packet.AsSlices())
		if err != nil {
			return n, &tcpip.ErrAborted{}
		}
		n++
	}
	return n, nil
}

// WritePacket writes outbound packets
func (e *DarwinEndpoint) WritePacket(pkt *stack.PacketBuffer) tcpip.Error {
	return nil
}
