package tunnel

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"time"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/conn"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/tunnel/nat"
)

// _udpSessionTimeout is the default timeout for each UDP session.
const (
	_udpSessionTimeout = 60 * time.Second
	defaultUDPTimeout  = 5 * time.Second
)

func (t *tunnel) handleUDPConn(packet adapter.PacketAdapter) {
	m := packet.Metadata()
	if !m.Valid() {
		//packet.Drop()
		log.Debugf("[Metadata] not valid: %#v", m)
		return
	}

	var fAddr netip.Addr

	if err := preHandleMetadata(m); err != nil {
		//packet.Drop()
		log.Debugf("[Metadata PreHandle] error: %s", err)
		return
	}

	key := packet.LocalAddr().String()
	natTable := t.natTable
	handle := func() bool {
		pc, proxy := natTable.Get(key)
		if pc != nil {
			if proxy != nil {
				proxy.UpdateWriteBack(packet)
			}
			_ = handleUDPToRemote(packet, pc, m)
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

		proxy := t.resolveMetadata(m)
		ctx, cancel := context.WithTimeout(context.Background(), defaultUDPTimeout)
		defer cancel()

		pc, err := proxy.ListenPacketContext(ctx, m.Pure())
		if err != nil {
			return
		}
		oAddrPort := m.AddrPort()

		writeBackProxy := nat.NewWriteBackProxy(packet)
		natTable.Set(key, pc, writeBackProxy)
		go t.handleUDPToLocal(writeBackProxy, pc, key, oAddrPort, fAddr)

		handle()

	}()

}

func handleUDPToRemote(packet adapter.UDPPacket, pc proxy.PacketConn, metadata *metadata.Metadata) error {
	addr := metadata.UDPAddr()
	if addr == nil {
		return errors.New("udp addr invalid")
	}

	if _, err := pc.WriteTo(packet.Data(), addr); err != nil {
		return err
	}
	// reset timeout
	_ = pc.SetReadDeadline(time.Now().Add(_udpSessionTimeout))

	return nil
}

func (t *tunnel) handleUDPToLocal(writeBack adapter.WriteBack, pc conn.EnhancePacketConn, key string, oAddrPort netip.AddrPort,
	fAddr netip.Addr) {
	defer func() {
		_ = pc.Close()
		t.closeAllLocalCoon(key)
		t.natTable.Delete(key)
	}()

	for {
		_ = pc.SetReadDeadline(time.Now().Add(_udpSessionTimeout))
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
