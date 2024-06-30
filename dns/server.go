package dns

import (
	"context"
	"errors"
	"net"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/common/sockopt"
	"github.com/lumavpn/luma/log"
	D "github.com/miekg/dns"
)

const dnsDefaultTTL uint32 = 600

type handler func(ctx *adapter.DNSContext, r *D.Msg) (*D.Msg, error)

// A Server defines parameters for running an DNS server
type Server struct {
	*D.Server
	address string
	handler handler
}

type ServerOptions struct {
	Addr     string
	Mapper   *ResolverEnhancer
	Resolver *Resolver
}

func (s *Server) ServeDNS(w D.ResponseWriter, r *D.Msg) {
	msg, err := handlerWithContext(context.Background(), s.handler, r)
	if err != nil {
		D.HandleFailed(w, r)
		return
	}
	msg.Compress = true
	w.WriteMsg(msg)
}

func handlerWithContext(ctx context.Context, handler handler, msg *D.Msg) (*D.Msg, error) {
	if len(msg.Question) == 0 {
		return nil, errors.New("at least one question is required")
	}

	return handler(adapter.NewDNSContext(ctx, msg), msg)
}

// NewServer creates a new DNS server
func NewServer(opts ServerOptions) (*Server, error) {
	addr := opts.Addr
	if addr == "" {
		return nil, errors.New("Missing address")
	}
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	p, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	err = sockopt.UDPReuseaddr(p)
	if err != nil {
		log.Warnf("Failed to Reuse UDP Address: %s", err)

		err = nil
	}

	handler := NewHandler(opts.Resolver, opts.Mapper)
	s := &Server{
		address: addr,
		handler: handler,
	}
	s.Server = &D.Server{Addr: addr, PacketConn: p, Handler: s}

	go func() {
		s.ActivateAndServe()
	}()

	log.Infof("DNS server listening at: %s", p.LocalAddr().String())
	return s, nil
}
