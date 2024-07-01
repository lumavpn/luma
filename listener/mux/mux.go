package mux

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/common/bufio"
	"github.com/lumavpn/luma/common/bufio/deadline"
	"github.com/lumavpn/luma/common/errors"
	M "github.com/lumavpn/luma/common/metadata"
	"github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/pool"
	"github.com/lumavpn/luma/conn"
	"github.com/lumavpn/luma/log"
	C "github.com/lumavpn/luma/metadata"
	smux "github.com/lumavpn/luma/mux"
	"github.com/lumavpn/luma/proxy/inbound"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/util"
)

const UDPTimeout = 5 * time.Minute

type ListenerConfig struct {
	Tunnel     adapter.TransportHandler
	Proto      proto.Proto
	Additions  []inbound.Option
	UDPTimeout time.Duration
	MuxOption  MuxOption
}

type MuxOption struct {
	Padding bool          `yaml:"padding" json:"padding,omitempty"`
	Brutal  BrutalOptions `yaml:"brutal" json:"brutal,omitempty"`
}

type BrutalOptions struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Up      string `yaml:"up" json:"up,omitempty"`
	Down    string `yaml:"down" json:"down,omitempty"`
}

type ListenerHandler struct {
	ListenerConfig
	muxService *smux.Service
}

func UpstreamMetadata(metadata M.Metadata) M.Metadata {
	return M.Metadata{
		Source:      metadata.Source,
		Destination: metadata.Destination,
	}
}

func NewListenerHandler(lc ListenerConfig) (h *ListenerHandler, err error) {
	h = &ListenerHandler{ListenerConfig: lc}
	h.muxService, err = smux.NewService(smux.ServiceOptions{
		NewStreamContext: func(ctx context.Context, conn net.Conn) context.Context {
			return ctx
		},
		Handler: h,
		Padding: lc.MuxOption.Padding,
		Brutal: smux.BrutalOptions{
			Enabled:    lc.MuxOption.Brutal.Enabled,
			SendBPS:    util.StringToBps(lc.MuxOption.Brutal.Up),
			ReceiveBPS: util.StringToBps(lc.MuxOption.Brutal.Down),
		},
	})
	return
}

func (h *ListenerHandler) IsSpecialFqdn(fqdn string) bool {
	switch fqdn {
	case smux.Destination.Fqdn:
		return true
	default:
		return false
	}
}

func (h *ListenerHandler) ParseSpecialFqdn(ctx context.Context, conn net.Conn, metadata M.Metadata) error {
	switch metadata.Destination.Fqdn {
	case smux.Destination.Fqdn:
		return h.muxService.NewConnection(ctx, conn, UpstreamMetadata(metadata))
	}
	return errors.New("not special fqdn")
}

func (h *ListenerHandler) NewConnection(ctx context.Context, c net.Conn, metadata M.Metadata) error {
	log.Debug("NewConnection called")
	if h.IsSpecialFqdn(metadata.Destination.Fqdn) {
		return h.ParseSpecialFqdn(ctx, c, metadata)
	}

	if deadline.NeedAdditionalReadDeadline(c) {
		c = conn.NewDeadlineConn(c) // conn from sing should check NeedAdditionalReadDeadline
	}

	cMetadata := &C.Metadata{
		Network: C.TCP,
		Proto:   h.Proto,
	}
	inbound.WithOptions(cMetadata, inbound.WithDstAddr(metadata.Destination), inbound.WithSrcAddr(metadata.Source), inbound.WithInAddr(c.LocalAddr()))
	inbound.WithOptions(cMetadata, getAdditions(ctx)...)
	inbound.WithOptions(cMetadata, h.Additions...)

	h.Tunnel.HandleTCP(adapter.NewTCPConn(c, cMetadata)) // this goroutine must exit after conn unused
	return nil
}

func (h *ListenerHandler) NewPacketConnection(ctx context.Context, conn network.PacketConn, metadata M.Metadata) error {
	defer func() { _ = conn.Close() }()
	log.Debug("NewPacketConnection called")
	mutex := sync.Mutex{}
	conn2 := bufio.NewNetPacketConn(conn) // a new interface to set nil in defer
	defer func() {
		mutex.Lock() // this goroutine must exit after all conn.WritePacket() is not running
		defer mutex.Unlock()
		conn2 = nil
	}()
	rwOptions := network.ReadWaitOptions{}
	readWaiter, isReadWaiter := bufio.CreatePacketReadWaiter(conn)
	if isReadWaiter {
		readWaiter.InitializeReadWaiter(rwOptions)
	}
	for {
		var (
			buff *pool.Buffer
			dest M.Socksaddr
			err  error
		)
		if isReadWaiter {
			buff, dest, err = readWaiter.WaitReadPacket()
		} else {
			buff = rwOptions.NewPacketBuffer()
			dest, err = conn.ReadPacket(buff)
			if buff != nil {
				rwOptions.PostReturn(buff)
			}
		}
		if err != nil {
			buff.Release()
			if ShouldIgnorePacketError(err) {
				break
			}
			return err
		}
		cPacket := &packet{
			conn:  &conn2,
			mutex: &mutex,
			rAddr: metadata.Source.UDPAddr(),
			lAddr: conn.LocalAddr(),
			buff:  buff,
		}

		cMetadata := &C.Metadata{
			Network: C.UDP,
			Proto:   h.Proto,
		}
		inbound.WithOptions(cMetadata, inbound.WithDstAddr(dest), inbound.WithSrcAddr(metadata.Source), inbound.WithInAddr(conn.LocalAddr()))
		inbound.WithOptions(cMetadata, getAdditions(ctx)...)
		inbound.WithOptions(cMetadata, h.Additions...)
		h.Tunnel.HandleUDP(adapter.NewPacketAdapter(cPacket, cMetadata))
	}
	return nil
}

func (h *ListenerHandler) NewError(ctx context.Context, err error) {
	log.Warnf("%s listener get error: %+v", h.Proto.String(), err)
}

func ShouldIgnorePacketError(err error) bool {
	// ignore simple error
	if errors.IsTimeout(err) || errors.IsClosed(err) || errors.IsCanceled(err) {
		return true
	}
	return false
}

type packet struct {
	conn  *network.NetPacketConn
	mutex *sync.Mutex
	rAddr net.Addr
	lAddr net.Addr
	buff  *pool.Buffer
}

func (c *packet) Data() []byte {
	return c.buff.Bytes()
}

// WriteBack wirtes UDP packet with source(ip, port) = `addr`
func (c *packet) WriteBack(b []byte, addr net.Addr) (n int, err error) {
	if addr == nil {
		err = errors.New("address is invalid")
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	conn := *c.conn
	if conn == nil {
		err = errors.New("writeBack to closed connection")
		return
	}

	buff := pool.NewPacket()
	defer buff.Release()
	n, err = buff.Write(b)
	if err != nil {
		return
	}

	err = conn.WritePacket(buff, M.ParseSocksAddrFromNet(addr))
	if err != nil {
		return
	}
	return
}

// LocalAddr returns the source IP/Port of UDP Packet
func (c *packet) LocalAddr() net.Addr {
	return c.rAddr
}

func (c *packet) Drop() {
	c.buff.Release()
}

func (c *packet) InAddr() net.Addr {
	return c.lAddr
}
