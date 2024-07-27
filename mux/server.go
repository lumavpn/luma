package mux

import (
	"context"
	"net"

	E "github.com/lumavpn/luma/common/errors"
	"github.com/lumavpn/luma/common/task"
	"github.com/lumavpn/luma/internal/debug"
	"github.com/lumavpn/luma/internal/version"
	"github.com/lumavpn/luma/log"
	"github.com/sagernet/sing/common/bufio"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

type ServiceHandler interface {
	N.TCPConnectionHandler
	N.UDPConnectionHandler
}

type Service struct {
	newStreamContext func(context.Context, net.Conn) context.Context
	handler          ServiceHandler
	padding          bool
	brutal           BrutalOptions
}

type ServiceOptions struct {
	NewStreamContext func(context.Context, net.Conn) context.Context
	Handler          ServiceHandler
	Padding          bool
	Brutal           BrutalOptions
}

func NewService(options ServiceOptions) (*Service, error) {
	if options.Brutal.Enabled && !BrutalAvailable && !version.Debug() {
		return nil, E.New("TCP Brutal is only supported on Linux")
	}
	return &Service{
		newStreamContext: options.NewStreamContext,
		handler:          options.Handler,
		padding:          options.Padding,
		brutal:           options.Brutal,
	}, nil
}

func (s *Service) NewConnection(ctx context.Context, conn net.Conn, metadata M.Metadata) error {
	request, err := ReadRequest(conn)
	if err != nil {
		return err
	}
	if request.Padding {
		conn = newPaddingConn(conn)
	} else if s.padding {
		return E.New("non-padded connection rejected")
	}
	session, err := newServerSession(conn, request.Protocol)
	if err != nil {
		return err
	}
	var group task.Group
	group.Append0(func(_ context.Context) error {
		var stream net.Conn
		for {
			stream, err = session.Accept()
			if err != nil {
				return err
			}
			streamCtx := s.newStreamContext(ctx, stream)
			go func() {
				hErr := s.newConnection(streamCtx, conn, stream, metadata)
				if hErr != nil {
					log.Error(E.Cause(hErr, "handle connection"))
				}
			}()
		}
	})
	group.Cleanup(func() {
		session.Close()
	})
	return group.Run(ctx)
}

func (s *Service) newConnection(ctx context.Context, sessionConn net.Conn, stream net.Conn, metadata M.Metadata) error {
	stream = &wrapStream{stream}
	request, err := ReadStreamRequest(stream)
	if err != nil {
		return E.Cause(err, "read multiplex stream request")
	}
	metadata.Destination = request.Destination
	if request.Network == N.NetworkTCP {
		conn := &serverConn{ExtendedConn: bufio.NewExtendedConn(stream)}
		if request.Destination.Fqdn == BrutalExchangeDomain {
			defer stream.Close()
			var clientReceiveBPS uint64
			clientReceiveBPS, err = ReadBrutalRequest(conn)
			if err != nil {
				return E.Cause(err, "read brutal request")
			}
			if !s.brutal.Enabled {
				err = WriteBrutalResponse(conn, 0, false, "brutal is not enabled by the server")
				if err != nil {
					return E.Cause(err, "write brutal response")
				}
				return nil
			}
			sendBPS := s.brutal.SendBPS
			if clientReceiveBPS < sendBPS {
				sendBPS = clientReceiveBPS
			}
			err = SetBrutalOptions(sessionConn, sendBPS)
			if err != nil {
				// ignore error in test
				if !debug.Enabled {
					err = WriteBrutalResponse(conn, 0, false, E.Cause(err, "enable TCP Brutal").Error())
					if err != nil {
						return E.Cause(err, "write brutal response")
					}
					return nil
				}
			}
			err = WriteBrutalResponse(conn, s.brutal.ReceiveBPS, true, "")
			if err != nil {
				return E.Cause(err, "write brutal response")
			}
			return nil
		}
		log.Infof("inbound multiplex connection to %v", metadata.Destination)
		s.handler.NewConnection(ctx, conn, metadata)
		stream.Close()
	} else {
		var packetConn N.PacketConn
		if !request.PacketAddr {
			log.Infof("inbound multiplex packet connection to %v", metadata.Destination)
			packetConn = &serverPacketConn{ExtendedConn: bufio.NewExtendedConn(stream), destination: request.Destination}
		} else {
			log.Info("inbound multiplex packet connection")
			packetConn = &serverPacketAddrConn{ExtendedConn: bufio.NewExtendedConn(stream)}
		}
		s.handler.NewPacketConnection(ctx, packetConn, metadata)
		stream.Close()
	}
	return nil
}
