package mux

import (
	"context"
	"net"

	M "github.com/lumavpn/luma/common/metadata"
	"github.com/lumavpn/luma/common/network"
)

type ServiceHandler interface {
	network.TCPConnectionHandler
	network.UDPConnectionHandler
}

type Service struct {
	opts *ServiceOptions
}

type ServiceOptions struct {
	NewStreamContext func(context.Context, net.Conn) context.Context
	Handler          ServiceHandler
	Padding          bool
}

// NewService returns a new instance of NewService
func NewService(options ServiceOptions) (*Service, error) {
	return &Service{&options}, nil
}

func (s *Service) NewConnection(ctx context.Context, conn net.Conn, metadata M.Metadata) error {
	return nil
}
