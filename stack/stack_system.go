package stack

import (
	"context"
	"net"
	"net/netip"
	"syscall"
	"time"

	"github.com/lumavpn/luma/common/buf"
	"github.com/lumavpn/luma/common/control"
	"github.com/lumavpn/luma/common/errors"
	M "github.com/lumavpn/luma/common/metadata"
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/udpnat"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/stack/internal/lumatcpip"
	"github.com/lumavpn/luma/util"
)

var ErrIncludeAllNetworks = errors.New("`system` and `mixed` stack are not available when `includeAllNetworks` is enabled.")

type System struct {
	tun                Tun
	tunName            string
	mtu                int
	handler            Handler
	inet4Prefixes      []netip.Prefix
	inet6Prefixes      []netip.Prefix
	inet4ServerAddress netip.Addr
	inet4Address       netip.Addr
	inet6ServerAddress netip.Addr
	inet6Address       netip.Addr
	broadcastAddr      netip.Addr
	udpTimeout         int64
	tcpListener        net.Listener
	tcpListener6       net.Listener
	tcpPort            uint16
	tcpPort6           uint16
	tcpNat             *TCPNat
	udpNat             *udpnat.Service[netip.AddrPort]
	bindInterface      bool
	interfaceFinder    control.InterfaceFinder
	frontHeadroom      int
	txChecksumOffload  bool
}

type Session struct {
	SourceAddress      netip.Addr
	DestinationAddress netip.Addr
	SourcePort         uint16
	DestinationPort    uint16
}

func NewSystem(options *Config) (Stack, error) {
	log.Debug("Creating new system stack")
	stack := &System{
		tun:             options.Tun,
		tunName:         options.TunOptions.Name,
		mtu:             int(options.TunOptions.MTU),
		udpTimeout:      options.UDPTimeout,
		handler:         options.Handler,
		inet4Prefixes:   options.TunOptions.Inet4Address,
		inet6Prefixes:   options.TunOptions.Inet6Address,
		broadcastAddr:   BroadcastAddr(options.TunOptions.Inet4Address),
		bindInterface:   options.ForwarderBindInterface,
		interfaceFinder: options.InterfaceFinder,
	}
	if len(options.TunOptions.Inet4Address) > 0 {
		if options.TunOptions.Inet4Address[0].Bits() == 32 {
			return nil, errors.New("need one more IPv4 address in first prefix for system stack")
		}
		stack.inet4ServerAddress = options.TunOptions.Inet4Address[0].Addr()
		stack.inet4Address = stack.inet4ServerAddress.Next()
	}
	if len(options.TunOptions.Inet6Address) > 0 {
		if options.TunOptions.Inet6Address[0].Bits() == 128 {
			return nil, errors.New("need one more IPv6 address in first prefix for system stack")
		}
		stack.inet6ServerAddress = options.TunOptions.Inet6Address[0].Addr()
		stack.inet6Address = stack.inet6ServerAddress.Next()
	}
	if !stack.inet4Address.IsValid() && !stack.inet6Address.IsValid() {
		return nil, errors.New("missing interface address")
	}
	return stack, nil
}

func (s *System) Close() error {
	return util.Close(
		s.tcpListener,
		s.tcpListener6,
	)
}

func (s *System) Start(ctx context.Context) error {
	err := s.start(ctx)
	if err != nil {
		return err
	}
	go s.tunLoop(ctx)
	return nil
}

func (s *System) start(ctx context.Context) error {
	err := fixWindowsFirewall()
	if err != nil {
		return errors.Cause(err, "fix windows firewall for system stack")
	}
	var listener net.ListenConfig
	if s.bindInterface {
		listener.Control = control.Append(listener.Control, func(network, address string, conn syscall.RawConn) error {
			bindErr := control.BindToInterface0(s.interfaceFinder, conn, network, address, s.tunName, -1, true)
			if bindErr != nil {
				log.Warnf("bind forwarder to interface: %v", bindErr)
			}
			return nil
		})
	}
	if s.inet4Address.IsValid() {
		tcpListener, err := listener.Listen(ctx, "tcp4", net.JoinHostPort(s.inet4ServerAddress.String(), "0"))
		if err != nil {
			return err
		}
		s.tcpListener = tcpListener
		s.tcpPort = M.SocksaddrFromNet(tcpListener.Addr()).Port
		go s.acceptLoop(ctx, tcpListener)
	}
	if s.inet6Address.IsValid() {
		tcpListener, err := listener.Listen(ctx, "tcp6", net.JoinHostPort(s.inet6ServerAddress.String(), "0"))
		if err != nil {
			return err
		}
		s.tcpListener6 = tcpListener
		s.tcpPort6 = M.SocksaddrFromNet(tcpListener.Addr()).Port
		go s.acceptLoop(ctx, tcpListener)
	}
	s.tcpNat = NewNat(ctx, time.Second*time.Duration(s.udpTimeout))
	s.udpNat = udpnat.New[netip.AddrPort](s.udpTimeout, s.handler)
	return nil
}

func (s *System) tunLoop(ctx context.Context) {
	if winTun, isWinTun := s.tun.(WinTun); isWinTun {
		s.wintunLoop(ctx, winTun)
		return
	}
	if linuxTUN, isLinuxTUN := s.tun.(LinuxTUN); isLinuxTUN {
		s.frontHeadroom = linuxTUN.FrontHeadroom()
		s.txChecksumOffload = linuxTUN.TXChecksumOffload()
		batchSize := linuxTUN.BatchSize()
		if batchSize > 1 {
			s.batchLoop(ctx, linuxTUN, batchSize)
			return
		}
	}
	packetBuffer := make([]byte, s.mtu+PacketOffset)
	for {
		n, err := s.tun.Read(packetBuffer)
		if err != nil {
			if errors.IsClosed(err) {
				return
			}
			log.Error(errors.Cause(err, "read packet"))
		}
		if n < lumatcpip.IPv4PacketMinLength {
			continue
		}
		rawPacket := packetBuffer[:n]
		packet := packetBuffer[PacketOffset:n]
		if s.processPacket(ctx, packet) {
			_, err = s.tun.Write(rawPacket)
			if err != nil {
				log.Error(errors.Cause(err, "write packet"))
			}
		}
	}
}

func (s *System) wintunLoop(ctx context.Context, winTun WinTun) {
	for {
		packet, release, err := winTun.ReadPacket()
		if err != nil {
			return
		}
		if len(packet) < lumatcpip.IPv4PacketMinLength {
			release()
			continue
		}
		if s.processPacket(ctx, packet) {
			_, err = winTun.Write(packet)
			if err != nil {
				log.Error(errors.Cause(err, "write packet"))
			}
		}
		release()
	}
}

func (s *System) batchLoop(ctx context.Context, linuxTUN LinuxTUN, batchSize int) {
	packetBuffers := make([][]byte, batchSize)
	writeBuffers := make([][]byte, batchSize)
	packetSizes := make([]int, batchSize)
	for i := range packetBuffers {
		packetBuffers[i] = make([]byte, s.mtu+s.frontHeadroom)
	}
	for {
		n, err := linuxTUN.BatchRead(packetBuffers, s.frontHeadroom, packetSizes)
		if err != nil {
			if errors.IsClosed(err) {
				return
			}
			log.Error(errors.Cause(err, "batch read packet"))
		}
		if n == 0 {
			continue
		}
		for i := 0; i < n; i++ {
			packetSize := packetSizes[i]
			if packetSize < lumatcpip.IPv4PacketMinLength {
				continue
			}
			packetBuffer := packetBuffers[i]
			packet := packetBuffer[s.frontHeadroom : s.frontHeadroom+packetSize]
			if s.processPacket(ctx, packet) {
				writeBuffers = append(writeBuffers, packetBuffer[:s.frontHeadroom+packetSize])
			}
		}
		if len(writeBuffers) > 0 {
			err = linuxTUN.BatchWrite(writeBuffers, s.frontHeadroom)
			if err != nil {
				log.Error(errors.Cause(err, "batch write packet"))
			}
			writeBuffers = writeBuffers[:0]
		}
	}
}

func (s *System) processPacket(ctx context.Context, packet []byte) bool {
	var (
		writeBack bool
		err       error
	)
	switch ipVersion := packet[0] >> 4; ipVersion {
	case 4:
		writeBack, err = s.processIPv4(ctx, packet)
	case 6:
		writeBack, err = s.processIPv6(ctx, packet)
	default:
		err = errors.New("ip: unknown version: ", ipVersion)
	}
	if err != nil {
		log.Error(err)
		return false
	}
	return writeBack
}

func (s *System) acceptLoop(ctx context.Context, listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		connPort := M.SocksaddrFromNet(conn.RemoteAddr()).Port
		session := s.tcpNat.LookupBack(connPort)
		if session == nil {
			log.Error(errors.New("unknown session with port ", connPort))
			continue
		}
		destination := M.SocksaddrFromNetIP(session.Destination)
		if destination.Addr.Is4() {
			for _, prefix := range s.inet4Prefixes {
				if prefix.Contains(destination.Addr) {
					destination.Addr = netip.AddrFrom4([4]byte{127, 0, 0, 1})
					break
				}
			}
		} else {
			for _, prefix := range s.inet6Prefixes {
				if prefix.Contains(destination.Addr) {
					destination.Addr = netip.AddrFrom16([16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
					break
				}
			}
		}
		go func() {
			_ = s.handler.NewConnection(ctx, conn, M.Metadata{
				Source:      M.SocksaddrFromNetIP(session.Source),
				Destination: destination,
			})
			if tcpConn, isTCPConn := conn.(*net.TCPConn); isTCPConn {
				_ = tcpConn.SetLinger(0)
			}
			_ = conn.Close()
		}()
	}
}

func (s *System) processIPv4(ctx context.Context, packet lumatcpip.IPv4Packet) (writeBack bool, err error) {
	writeBack = true
	destination := packet.DestinationIP()
	if destination == s.broadcastAddr || !destination.IsGlobalUnicast() {
		return
	}
	switch packet.Protocol() {
	case lumatcpip.TCP:
		err = s.processIPv4TCP(packet, packet.Payload())
	case lumatcpip.UDP:
		writeBack = false
		err = s.processIPv4UDP(ctx, packet, packet.Payload())
	case lumatcpip.ICMP:
		err = s.processIPv4ICMP(packet, packet.Payload())
	}
	return
}

func (s *System) processIPv6(ctx context.Context, packet lumatcpip.IPv6Packet) (writeBack bool, err error) {
	writeBack = true
	if !packet.DestinationIP().IsGlobalUnicast() {
		return
	}
	switch packet.Protocol() {
	case lumatcpip.TCP:
		err = s.processIPv6TCP(packet, packet.Payload())
	case lumatcpip.UDP:
		writeBack = false
		err = s.processIPv6UDP(ctx, packet, packet.Payload())
	case lumatcpip.ICMPv6:
		err = s.processIPv6ICMP(packet, packet.Payload())
	}
	return
}

func (s *System) processIPv4TCP(packet lumatcpip.IPv4Packet, header lumatcpip.TCPPacket) error {
	source := netip.AddrPortFrom(packet.SourceIP(), header.SourcePort())
	destination := netip.AddrPortFrom(packet.DestinationIP(), header.DestinationPort())
	if !destination.Addr().IsGlobalUnicast() {
		return nil
	} else if source.Addr() == s.inet4ServerAddress && source.Port() == s.tcpPort {
		session := s.tcpNat.LookupBack(destination.Port())
		if session == nil {
			return errors.New("ipv4: tcp: session not found: ", destination.Port())
		}
		packet.SetSourceIP(session.Destination.Addr())
		header.SetSourcePort(session.Destination.Port())
		packet.SetDestinationIP(session.Source.Addr())
		header.SetDestinationPort(session.Source.Port())
	} else {
		natPort := s.tcpNat.Lookup(source, destination)
		packet.SetSourceIP(s.inet4Address)
		header.SetSourcePort(natPort)
		packet.SetDestinationIP(s.inet4ServerAddress)
		header.SetDestinationPort(s.tcpPort)
	}
	if !s.txChecksumOffload {
		header.ResetChecksum(packet.PseudoSum())
		packet.ResetChecksum()
	} else {
		header.OffloadChecksum()
		packet.ResetChecksum()
	}
	return nil
}

func (s *System) processIPv6TCP(packet lumatcpip.IPv6Packet, header lumatcpip.TCPPacket) error {
	source := netip.AddrPortFrom(packet.SourceIP(), header.SourcePort())
	destination := netip.AddrPortFrom(packet.DestinationIP(), header.DestinationPort())
	if !destination.Addr().IsGlobalUnicast() {
		return nil
	} else if source.Addr() == s.inet6ServerAddress && source.Port() == s.tcpPort6 {
		session := s.tcpNat.LookupBack(destination.Port())
		if session == nil {
			return errors.New("ipv6: tcp: session not found: ", destination.Port())
		}
		packet.SetSourceIP(session.Destination.Addr())
		header.SetSourcePort(session.Destination.Port())
		packet.SetDestinationIP(session.Source.Addr())
		header.SetDestinationPort(session.Source.Port())
	} else {
		natPort := s.tcpNat.Lookup(source, destination)
		packet.SetSourceIP(s.inet6Address)
		header.SetSourcePort(natPort)
		packet.SetDestinationIP(s.inet6ServerAddress)
		header.SetDestinationPort(s.tcpPort6)
	}
	if !s.txChecksumOffload {
		header.ResetChecksum(packet.PseudoSum())
	} else {
		header.OffloadChecksum()
	}
	return nil
}

func (s *System) processIPv4UDP(ctx context.Context, packet lumatcpip.IPv4Packet, header lumatcpip.UDPPacket) error {
	if packet.Flags()&lumatcpip.FlagMoreFragment != 0 {
		return errors.New("ipv4: fragment dropped")
	}
	if packet.FragmentOffset() != 0 {
		return errors.New("ipv4: udp: fragment dropped")
	}
	if !header.Valid() {
		return errors.New("ipv4: udp: invalid packet")
	}
	source := netip.AddrPortFrom(packet.SourceIP(), header.SourcePort())
	destination := netip.AddrPortFrom(packet.DestinationIP(), header.DestinationPort())
	if !destination.Addr().IsGlobalUnicast() {
		return nil
	}
	data := buf.As(header.Payload())
	if data.Len() == 0 {
		return nil
	}
	metadata := M.Metadata{
		Source:      M.SocksaddrFromNetIP(source),
		Destination: M.SocksaddrFromNetIP(destination),
	}
	s.udpNat.NewPacket(ctx, source, data.ToOwned(), metadata, func(natConn N.PacketConn) N.PacketWriter {
		headerLen := packet.HeaderLen() + lumatcpip.UDPHeaderSize
		headerCopy := make([]byte, headerLen)
		copy(headerCopy, packet[:headerLen])
		return &systemUDPPacketWriter4{
			s.tun,
			s.frontHeadroom + PacketOffset,
			headerCopy,
			source,
			s.txChecksumOffload,
		}
	})
	return nil
}

func (s *System) processIPv6UDP(ctx context.Context, packet lumatcpip.IPv6Packet, header lumatcpip.UDPPacket) error {
	if !header.Valid() {
		return errors.New("ipv6: udp: invalid packet")
	}
	source := netip.AddrPortFrom(packet.SourceIP(), header.SourcePort())
	destination := netip.AddrPortFrom(packet.DestinationIP(), header.DestinationPort())
	if !destination.Addr().IsGlobalUnicast() {
		return nil
	}
	data := buf.As(header.Payload())
	if data.Len() == 0 {
		return nil
	}
	metadata := M.Metadata{
		Source:      M.SocksaddrFromNetIP(source),
		Destination: M.SocksaddrFromNetIP(destination),
	}
	s.udpNat.NewPacket(ctx, source, data.ToOwned(), metadata, func(natConn N.PacketConn) N.PacketWriter {
		headerLen := len(packet) - int(header.Length()) + lumatcpip.UDPHeaderSize
		headerCopy := make([]byte, headerLen)
		copy(headerCopy, packet[:headerLen])
		return &systemUDPPacketWriter6{
			s.tun,
			s.frontHeadroom + PacketOffset,
			headerCopy,
			source,
			s.txChecksumOffload,
		}
	})
	return nil
}

func (s *System) processIPv4ICMP(packet lumatcpip.IPv4Packet, header lumatcpip.ICMPPacket) error {
	if header.Type() != lumatcpip.ICMPTypePingRequest || header.Code() != 0 {
		return nil
	}
	header.SetType(lumatcpip.ICMPTypePingResponse)
	sourceAddress := packet.SourceIP()
	packet.SetSourceIP(packet.DestinationIP())
	packet.SetDestinationIP(sourceAddress)
	header.ResetChecksum()
	packet.ResetChecksum()
	return nil
}

func (s *System) processIPv6ICMP(packet lumatcpip.IPv6Packet, header lumatcpip.ICMPv6Packet) error {
	if header.Type() != lumatcpip.ICMPv6EchoRequest || header.Code() != 0 {
		return nil
	}
	header.SetType(lumatcpip.ICMPv6EchoReply)
	sourceAddress := packet.SourceIP()
	packet.SetSourceIP(packet.DestinationIP())
	packet.SetDestinationIP(sourceAddress)
	header.ResetChecksum(packet.PseudoSum())
	packet.ResetChecksum()
	return nil
}

type systemUDPPacketWriter4 struct {
	tun               Tun
	frontHeadroom     int
	header            []byte
	source            netip.AddrPort
	txChecksumOffload bool
}

func (w *systemUDPPacketWriter4) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	newPacket := buf.NewSize(w.frontHeadroom + len(w.header) + buffer.Len())
	defer newPacket.Release()
	newPacket.Resize(w.frontHeadroom, 0)
	newPacket.Write(w.header)
	newPacket.Write(buffer.Bytes())
	ipHdr := lumatcpip.IPv4Packet(newPacket.Bytes())
	ipHdr.SetTotalLength(uint16(newPacket.Len()))
	ipHdr.SetDestinationIP(ipHdr.SourceIP())
	ipHdr.SetSourceIP(destination.Addr)
	udpHdr := lumatcpip.UDPPacket(ipHdr.Payload())
	udpHdr.SetDestinationPort(udpHdr.SourcePort())
	udpHdr.SetSourcePort(destination.Port)
	udpHdr.SetLength(uint16(buffer.Len() + lumatcpip.UDPHeaderSize))
	if !w.txChecksumOffload {
		udpHdr.ResetChecksum(ipHdr.PseudoSum())
		ipHdr.ResetChecksum()
	} else {
		udpHdr.OffloadChecksum()
		ipHdr.ResetChecksum()
	}
	if PacketOffset > 0 {
		newPacket.ExtendHeader(PacketOffset)[3] = syscall.AF_INET
	} else {
		newPacket.Advance(-w.frontHeadroom)
	}
	return util.Error(w.tun.Write(newPacket.Bytes()))
}

type systemUDPPacketWriter6 struct {
	tun               Tun
	frontHeadroom     int
	header            []byte
	source            netip.AddrPort
	txChecksumOffload bool
}

func (w *systemUDPPacketWriter6) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	newPacket := buf.NewSize(w.frontHeadroom + len(w.header) + buffer.Len())
	defer newPacket.Release()
	newPacket.Resize(w.frontHeadroom, 0)
	newPacket.Write(w.header)
	newPacket.Write(buffer.Bytes())
	ipHdr := lumatcpip.IPv6Packet(newPacket.Bytes())
	udpLen := uint16(lumatcpip.UDPHeaderSize + buffer.Len())
	ipHdr.SetPayloadLength(udpLen)
	ipHdr.SetDestinationIP(ipHdr.SourceIP())
	ipHdr.SetSourceIP(destination.Addr)
	udpHdr := lumatcpip.UDPPacket(ipHdr.Payload())
	udpHdr.SetDestinationPort(udpHdr.SourcePort())
	udpHdr.SetSourcePort(destination.Port)
	udpHdr.SetLength(udpLen)
	if !w.txChecksumOffload {
		udpHdr.ResetChecksum(ipHdr.PseudoSum())
	} else {
		udpHdr.OffloadChecksum()
	}
	if PacketOffset > 0 {
		newPacket.ExtendHeader(PacketOffset)[3] = syscall.AF_INET6
	} else {
		newPacket.Advance(-w.frontHeadroom)
	}
	return util.Error(w.tun.Write(newPacket.Bytes()))
}
