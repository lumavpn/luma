package tunnel

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/lumavpn/luma/adapter"
	C "github.com/lumavpn/luma/common"
	N "github.com/lumavpn/luma/common/net"
	"github.com/lumavpn/luma/component/resolver"
	"github.com/lumavpn/luma/log"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	P "github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/tunnel/nat"
	"github.com/lumavpn/luma/tunnel/sniffer"
	"github.com/lumavpn/luma/tunnel/statistic"
)

const (
	// MaxSegmentSize is the largest possible UDP datagram size.
	maxSegmentSize = (1 << 16) - 1

	// _udpRelayBufferSize is the default size for UDP packets relay.
	_udpRelayBufferSize = 16 << 10

	// default timeout for UDP session
	udpTimeout = 60 * time.Second
)

var (
	// udpSessionTimeout is the default timeout for each UDP session.
	udpSessionTimeout = 60 * time.Second
	// default timeout for UDP session
	defaultUDPTimeout = 5 * time.Second
)

func (t *tunnel) handleUDPConn(packet adapter.PacketAdapter) {
	if !t.isHandle(packet.Metadata().Type) {
		packet.Drop()
		return
	}

	metadata := packet.Metadata()
	if !metadata.Valid() {
		packet.Drop()
		log.Warnf("[Metadata] not valid: %#v", metadata)
		return
	}

	// make a fAddr if request ip is fakeip
	var fAddr netip.Addr
	if resolver.IsExistFakeIP(metadata.DstIP) {
		fAddr = metadata.DstIP
	}

	if err := preHandleMetadata(metadata); err != nil {
		packet.Drop()
		log.Debugf("[Metadata PreHandle] error: %s", err)
		return
	}

	if sniffer.Dispatcher.Enable() && t.sniffingEnable {
		sniffer.Dispatcher.UDPSniff(packet)
	}

	// local resolve UDP dns
	if !metadata.Resolved() {
		ip, err := resolver.ResolveIP(context.Background(), metadata.Host)
		if err != nil {
			return
		}
		log.Debugf("Resolved host %s to %v", metadata.Host, ip)
		metadata.DstIP = ip
	}

	key := packet.LocalAddr().String()
	natTable := t.natTable
	handle := func() bool {
		pc, proxy := natTable.Get(key)
		if pc != nil {
			if proxy != nil {
				proxy.UpdateWriteBack(packet)
			}
			_ = handleUDPToRemote(packet, pc, metadata)
			return true
		}
		return false
	}

	if handle() {
		packet.Drop()
		return
	}
	cond, loaded := natTable.GetOrCreateLock(key)

	go func() {
		defer packet.Drop()

		if loaded {
			cond.L.Lock()
			cond.Wait()
			handle()
			cond.L.Unlock()
			return
		}

		defer func() {
			natTable.DeleteLock(key)
			cond.Broadcast()
		}()

		proxy, rule, err := t.proxyDialer.ResolveMetadata(metadata)
		if err != nil {
			log.Warnf("[UDP] Parse metadata failed: %s", err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), C.DefaultUDPTimeout)
		defer cancel()
		rawPc, err := retry(ctx, func(ctx context.Context) (P.PacketConn, error) {
			return proxy.ListenPacketContext(ctx, metadata.Pure())
		}, func(err error) {
			if rule == nil {
				log.Warnf(
					"[UDP] dial %s %s --> %s error: %s",
					proxy.Name(),
					metadata.SourceDetail(),
					metadata.RemoteAddress(),
					err.Error(),
				)
			} else {
				log.Warnf("[UDP] dial %s (match %s/%s) %s --> %s error: %s", proxy.Name(), rule.RuleType().String(), rule.Payload(), metadata.SourceDetail(), metadata.RemoteAddress(), err.Error())
			}
		})
		if err != nil {
			return
		}
		pc := statistic.NewUDPTracker(rawPc, statistic.DefaultManager, metadata, rule, 0, 0, true)
		mode := t.mode
		switch true {
		case metadata.SpecialProxy != "":
			log.Infof("[UDP] %s --> %s using %s", metadata.SourceDetail(), metadata.RemoteAddress(), metadata.SpecialProxy)
		case rule != nil:
			if rule.Payload() != "" {
				log.Infof("[UDP] %s --> %s match %s using %s", metadata.SourceDetail(), metadata.RemoteAddress(), fmt.Sprintf("%s(%s)", rule.RuleType().String(), rule.Payload()), rawPc.Chains().String())
				if rawPc.Chains().Last() == "REJECT-DROP" {
					pc.Close()
					return
				}
			} else {
				log.Infof("[UDP] %s --> %s match %s using %s", metadata.SourceDetail(), metadata.RemoteAddress(), rule.Payload(), rawPc.Chains().String())
			}
		case mode == C.Global:
			log.Infof("[UDP] %s --> %s using GLOBAL", metadata.SourceDetail(), metadata.RemoteAddress())
		case mode == C.Select && proxy != nil:
			log.Infof("[UDP] %s --> %s using %s", metadata.SourceDetail(), metadata.RemoteAddress(), proxy.Addr())
		default:
			log.Infof("[UDP] %s --> %s doesn't match any rule using DIRECT", metadata.SourceDetail(), metadata.RemoteAddress())
		}

		oAddrPort := metadata.AddrPort()

		writeBackProxy := nat.NewWriteBackProxy(packet)
		natTable.Set(key, pc, writeBackProxy)
		go t.handleUDPToLocal(writeBackProxy, pc, key, oAddrPort, fAddr)

		handle()
	}()
}

func handleUDPToRemote(packet adapter.UDPPacket, pc proxy.PacketConn, metadata *M.Metadata) error {
	addr := metadata.UDPAddr()
	if addr == nil {
		return errors.New("udp addr invalid")
	}

	if _, err := pc.WriteTo(packet.Data(), addr); err != nil {
		return err
	}
	// reset timeout
	_ = pc.SetReadDeadline(time.Now().Add(udpTimeout))

	return nil
}

func (t *tunnel) handleUDPToLocal(writeBack adapter.WriteBack, pc N.EnhancePacketConn, key string, oAddrPort netip.AddrPort, fAddr netip.Addr) {
	defer func() {
		_ = pc.Close()
		t.closeAllLocalCoon(key)
		t.natTable.Delete(key)
	}()

	for {
		_ = pc.SetReadDeadline(time.Now().Add(udpTimeout))
		data, put, from, err := pc.WaitReadFrom()
		if err != nil {
			return
		}

		fromUDPAddr, isUDPAddr := from.(*net.UDPAddr)
		if !isUDPAddr {
			fromUDPAddr = net.UDPAddrFromAddrPort(oAddrPort) // oAddrPort was Unmapped
			log.Warnf("server return a [%T](%s) which isn't a *net.UDPAddr, force replace to (%s), this may be caused by a wrongly implemented server", from, from, oAddrPort)
		} else if fromUDPAddr == nil {
			fromUDPAddr = net.UDPAddrFromAddrPort(oAddrPort) // oAddrPort was Unmapped
			log.Warnf("server return a nil *net.UDPAddr, force replace to (%s), this may be caused by a wrongly implemented server", oAddrPort)
		} else {
			_fromUDPAddr := *fromUDPAddr
			fromUDPAddr = &_fromUDPAddr // make a copy
			if fromAddr, ok := netip.AddrFromSlice(fromUDPAddr.IP); ok {
				fromAddr = fromAddr.Unmap()
				if fAddr.IsValid() && (oAddrPort.Addr() == fromAddr) { // oAddrPort was Unmapped
					fromAddr = fAddr.Unmap()
				}
				fromUDPAddr.IP = fromAddr.AsSlice()
				if fromAddr.Is4() {
					fromUDPAddr.Zone = "" // only ipv6 can have the zone
				}
			}
		}

		_, err = writeBack.WriteBack(data, fromUDPAddr)
		if put != nil {
			put()
		}
		if err != nil {
			return
		}
	}
}

func (t *tunnel) closeAllLocalCoon(lAddr string) {
	t.natTable.RangeForLocalConn(lAddr, func(key string, value *net.UDPConn) bool {
		conn := value

		conn.Close()
		log.Debugf("Closing TProxy local conn... lAddr=%s rAddr=%s", lAddr, key)
		return true
	})
}
