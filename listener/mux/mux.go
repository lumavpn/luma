package mux

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/common/buf"
	"github.com/lumavpn/luma/common/bufio"
	"github.com/lumavpn/luma/common/metadata"
	"github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/log"
	M "github.com/lumavpn/luma/metadata"
	smux "github.com/lumavpn/luma/mux"
	"github.com/lumavpn/luma/proxy/inbound"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/util"
)

const UDPTimeout = 5 * time.Minute

type Listener struct {
	Options
	muxService *smux.Service
}

type Options struct {
	Tunnel     adapter.TransportHandler
	Type       proto.Proto
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

func NewListener(opts Options) (*Listener, error) {
	h := &Listener{Options: opts}
	var err error
	h.muxService, err = smux.NewService(smux.ServiceOptions{
		NewStreamContext: func(ctx context.Context, conn net.Conn) context.Context {
			return ctx
		},
		Handler: h,
		Padding: opts.MuxOption.Padding,
		Brutal: smux.BrutalOptions{
			Enabled:    opts.MuxOption.Brutal.Enabled,
			SendBPS:    util.StringToBps(opts.MuxOption.Brutal.Up),
			ReceiveBPS: util.StringToBps(opts.MuxOption.Brutal.Down),
		},
	})
	return h, err
}

func UpstreamMetadata(m metadata.Metadata) metadata.Metadata {
	return metadata.Metadata{
		Source:      m.Source,
		Destination: m.Destination,
	}
}

func (h *Listener) IsSpecialFqdn(fqdn string) bool {
	switch fqdn {
	case smux.Destination.Fqdn:
		return true
	default:
		return false
	}
}

func (h *Listener) ParseSpecialFqdn(ctx context.Context, conn net.Conn, metadata metadata.Metadata) error {
	switch metadata.Destination.Fqdn {
	case smux.Destination.Fqdn:
		return h.muxService.NewConnection(ctx, conn, UpstreamMetadata(metadata))
	}
	return errors.New("not special fqdn")
}

func (h *Listener) NewConnection(ctx context.Context, conn net.Conn, metadata metadata.Metadata) error {
	log.Debug("NewConnection called")
	if h.IsSpecialFqdn(metadata.Destination.Fqdn) {
		return h.ParseSpecialFqdn(ctx, conn, metadata)
	}

	cMetadata := &M.Metadata{
		Network: M.TCP,
		Type:    h.Type,
	}
	inbound.WithOptions(cMetadata, inbound.WithDstAddr(metadata.Destination), inbound.WithSrcAddr(metadata.Source), inbound.WithInAddr(conn.LocalAddr()))
	inbound.WithOptions(cMetadata, getAdditions(ctx)...)
	inbound.WithOptions(cMetadata, h.Additions...)

	h.Tunnel.HandleTCPConn(conn, cMetadata)
	return nil
}

func (h *Listener) NewPacketConnection(ctx context.Context, conn network.PacketConn, m metadata.Metadata) error {
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
			buff *buf.Buffer
			dest metadata.Socksaddr
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
			rAddr: m.Source.UDPAddr(),
			lAddr: conn.LocalAddr(),
			buff:  buff,
		}

		cMetadata := &M.Metadata{
			Network: M.UDP,
			Type:    h.Type,
		}
		inbound.WithOptions(cMetadata, inbound.WithDstAddr(dest), inbound.WithSrcAddr(m.Source), inbound.WithInAddr(conn.LocalAddr()))
		inbound.WithOptions(cMetadata, getAdditions(ctx)...)
		inbound.WithOptions(cMetadata, h.Additions...)
		h.Tunnel.HandleUDPPacket(cPacket, cMetadata)
	}
	return nil
}

func (h *Listener) NewError(ctx context.Context, err error) {
	log.Warnf("%s listener get error: %+v", h.Type.String(), err)
}
