package local

import (
	"fmt"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/listener/socks"
	"github.com/lumavpn/luma/log"
)

type SocksOption struct {
	*BaseOption
	Addr string `inbound:"addr,omitempty"`
	Port int    `inbound:"port,omitempty"`
	UDP  bool   `inbound:"udp,omitempty"`
}

type SocksServer struct {
	*BaseServer
	udp bool
	stl *socks.Listener
	sul *socks.UDPListener
}

func NewSocks(opts *SocksOption) (*SocksServer, error) {
	base, err := NewBase(&BaseOption{
		Addr: opts.Addr,
	})
	if err != nil {
		return nil, err
	}
	return &SocksServer{
		BaseServer: base,
		udp:        opts.UDP,
	}, nil
}

func (s *SocksServer) Stop() error {
	var err error
	if s.stl != nil {
		if tcpErr := s.stl.Close(); tcpErr != nil {
			err = tcpErr
		}
	}
	if s.udp && s.sul != nil {
		if udpErr := s.sul.Close(); udpErr != nil {
			if err == nil {
				err = udpErr
			} else {
				return fmt.Errorf("close tcp err: %s, close udp err: %s", err.Error(), udpErr.Error())
			}
		}
	}

	return err
}

func (s *SocksServer) Address() string {
	return s.stl.Address()
}

func (s *SocksServer) Start(tunnel adapter.TransportHandler) error {
	var err error
	if s.stl, err = socks.New(s.RawAddress(), tunnel, s.Additions()...); err != nil {
		return err
	}
	if s.udp {
		if s.sul, err = socks.NewUDP(s.RawAddress(), tunnel, s.Additions()...); err != nil {
			return err
		}
	}

	log.Infof("SOCKS[%s] proxy listening at: %s", s.Name(), s.Address())
	return nil
}
