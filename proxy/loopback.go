package proxy

import (
	"errors"
	"fmt"
	"net/netip"

	"github.com/lumavpn/luma/component/iface"
	M "github.com/lumavpn/luma/metadata"

	"github.com/puzpuzpuz/xsync/v3"
)

var ErrReject = errors.New("reject loopback connection")

type Detector struct {
	connMap       *xsync.MapOf[netip.AddrPort, struct{}]
	packetConnMap *xsync.MapOf[uint16, struct{}]
}

func NewDetector() *Detector {
	return &Detector{
		connMap:       xsync.NewMapOf[netip.AddrPort, struct{}](),
		packetConnMap: xsync.NewMapOf[uint16, struct{}](),
	}
}

func (l *Detector) NewConn(conn Conn) Conn {
	metadata := M.Metadata{}
	if metadata.SetRemoteAddr(conn.LocalAddr()) != nil {
		return conn
	}
	connAddr := metadata.AddrPort()
	if !connAddr.IsValid() {
		return conn
	}
	l.connMap.Store(connAddr, struct{}{})
	return NewCloseCallbackConn(conn, func() {
		l.connMap.Delete(connAddr)
	})
}

func (l *Detector) NewPacketConn(conn PacketConn) PacketConn {
	metadata := M.Metadata{}
	if metadata.SetRemoteAddr(conn.LocalAddr()) != nil {
		return conn
	}
	connAddr := metadata.AddrPort()
	if !connAddr.IsValid() {
		return conn
	}
	port := connAddr.Port()
	l.packetConnMap.Store(port, struct{}{})
	return NewCloseCallbackPacketConn(conn, func() {
		l.packetConnMap.Delete(port)
	})
}

func (l *Detector) CheckConn(metadata *M.Metadata) error {
	connAddr := metadata.SourceAddrPort()
	if !connAddr.IsValid() {
		return nil
	}
	if _, ok := l.connMap.Load(connAddr); ok {
		return fmt.Errorf("%w to: %s", ErrReject, metadata.DestinationAddress())
	}
	return nil
}

func (l *Detector) CheckPacketConn(metadata *M.Metadata) error {
	connAddr := metadata.SourceAddrPort()
	if !connAddr.IsValid() {
		return nil
	}

	isLocalIp, err := iface.IsLocalIp(connAddr.Addr())
	if err != nil {
		return err
	}
	if !isLocalIp && !connAddr.Addr().IsLoopback() {
		return nil
	}

	if _, ok := l.packetConnMap.Load(connAddr.Port()); ok {
		return fmt.Errorf("%w to: %s", ErrReject, metadata.DestinationAddress())
	}
	return nil
}
