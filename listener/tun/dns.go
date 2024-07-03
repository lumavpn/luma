package tun

import (
	"context"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/lumavpn/luma/component/resolver"
	"github.com/lumavpn/luma/listener/mux"
	"github.com/lumavpn/luma/log"

	"github.com/lumavpn/luma/common/buf"
	"github.com/lumavpn/luma/common/bufio"
	M "github.com/lumavpn/luma/common/metadata"
	"github.com/lumavpn/luma/common/network"
)

type ListenerHandler struct {
	*mux.ListenerHandler
	DnsAdds []netip.AddrPort
}

func (h *ListenerHandler) ShouldHijackDns(targetAddr netip.AddrPort) bool {
	if targetAddr.Addr().IsLoopback() && targetAddr.Port() == 53 { // cause by system stack
		return true
	}
	for _, addrPort := range h.DnsAdds {
		if addrPort == targetAddr || (addrPort.Addr().IsUnspecified() && targetAddr.Port() == 53) {
			return true
		}
	}
	return false
}

func (h *ListenerHandler) NewConnection(ctx context.Context, conn net.Conn, metadata M.Metadata) error {
	if h.ShouldHijackDns(metadata.Destination.AddrPort()) {
		log.Debugf("[DNS] hijack tcp:%s", metadata.Destination.String())
		return resolver.RelayDnsConn(ctx, conn, resolver.DefaultDnsReadTimeout)
	}
	return h.ListenerHandler.NewConnection(ctx, conn, metadata)
}

func (h *ListenerHandler) NewPacketConnection(ctx context.Context, conn network.PacketConn, metadata M.Metadata) error {
	if h.ShouldHijackDns(metadata.Destination.AddrPort()) {
		log.Debugf("[DNS] hijack udp:%s from %s", metadata.Destination.String(), metadata.Source.String())
		defer func() { _ = conn.Close() }()
		mutex := sync.Mutex{}
		conn2 := conn // a new interface to set nil in defer
		defer func() {
			mutex.Lock() // this goroutine must exit after all conn.WritePacket() is not running
			defer mutex.Unlock()
			conn2 = nil
		}()
		rwOptions := network.ReadWaitOptions{
			FrontHeadroom: network.CalculateFrontHeadroom(conn),
			RearHeadroom:  network.CalculateRearHeadroom(conn),
			MTU:           resolver.SafeDnsPacketSize,
		}
		readWaiter, isReadWaiter := bufio.CreatePacketReadWaiter(conn)
		if isReadWaiter {
			readWaiter.InitializeReadWaiter(rwOptions)
		}
		for {
			var (
				readBuff *buf.Buffer
				dest     M.Socksaddr
				err      error
			)
			_ = conn.SetReadDeadline(time.Now().Add(resolver.DefaultDnsReadTimeout))
			readBuff = nil // clear last loop status, avoid repeat release
			if isReadWaiter {
				readBuff, dest, err = readWaiter.WaitReadPacket()
			} else {
				readBuff = rwOptions.NewPacketBuffer()
				dest, err = conn.ReadPacket(readBuff)
				if readBuff != nil {
					rwOptions.PostReturn(readBuff)
				}
			}
			if err != nil {
				if readBuff != nil {
					readBuff.Release()
				}
				if mux.ShouldIgnorePacketError(err) {
					break
				}
				return err
			}
			go func() {
				ctx, cancel := context.WithTimeout(ctx, resolver.DefaultDnsRelayTimeout)
				defer cancel()
				inData := readBuff.Bytes()
				writeBuff := readBuff
				writeBuff.Resize(writeBuff.Start(), 0)
				if len(writeBuff.FreeBytes()) < resolver.SafeDnsPacketSize { // only create a new buffer when space don't enough
					writeBuff = rwOptions.NewPacketBuffer()
				}
				msg, err := resolver.RelayDnsPacket(ctx, inData, writeBuff.FreeBytes())
				if writeBuff != readBuff {
					readBuff.Release()
				}
				if err != nil {
					writeBuff.Release()
					return
				}
				writeBuff.Truncate(len(msg))
				mutex.Lock()
				defer mutex.Unlock()
				conn := conn2
				if conn == nil {
					writeBuff.Release()
					return
				}
				err = conn.WritePacket(writeBuff, dest) // WritePacket will release writeBuff
				if err != nil {
					writeBuff.Release()
					return
				}
			}()
		}
		return nil
	}
	return h.ListenerHandler.NewPacketConnection(ctx, conn, metadata)
}
