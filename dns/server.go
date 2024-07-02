package dns

import (
	"context"
	"errors"
	"net"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/common/sockopt"
	"github.com/lumavpn/luma/internal/features"
	"github.com/lumavpn/luma/log"

	D "github.com/miekg/dns"
)

var (
	address string
	server  = &Server{}

	dnsDefaultTTL uint32 = 600
)

type Server struct {
	*D.Server
	handler handler
}

// ServeDNS implement D.Handler ServeDNS
func (s *Server) ServeDNS(w D.ResponseWriter, r *D.Msg) {
	msg, err := handlerWithContext(context.Background(), s.handler, r)
	if err != nil {
		D.HandleFailed(w, r)
		return
	}
	msg.Compress = true
	w.WriteMsg(msg)
}

func handlerWithContext(stdCtx context.Context, handler handler, msg *D.Msg) (*D.Msg, error) {
	if len(msg.Question) == 0 {
		return nil, errors.New("at least one question is required")
	}

	ctx := adapter.NewDNSContext(stdCtx, msg)
	return handler(ctx, msg)
}

func (s *Server) SetHandler(handler handler) {
	s.handler = handler
}

func ReCreateServer(addr string, resolver *Resolver, mapper *ResolverEnhancer) {
	if features.CMFA {
		UpdateIsolateHandler(resolver, mapper)
	}

	if addr == address && resolver != nil {
		handler := NewHandler(resolver, mapper)
		server.SetHandler(handler)
		return
	}

	if server.Server != nil {
		server.Shutdown()
		server = &Server{}
		address = ""
	}

	if addr == "" {
		return
	}

	var err error
	defer func() {
		if err != nil {
			log.Errorf("Start DNS server error: %s", err.Error())
		}
	}()

	_, port, err := net.SplitHostPort(addr)
	if port == "0" || port == "" || err != nil {
		return
	}

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return
	}

	p, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return
	}

	err = sockopt.UDPReuseaddr(p)
	if err != nil {
		log.Warnf("Failed to Reuse UDP Address: %s", err)

		err = nil
	}

	address = addr
	handler := NewHandler(resolver, mapper)
	server = &Server{handler: handler}
	server.Server = &D.Server{Addr: addr, PacketConn: p, Handler: server}

	go func() {
		server.ActivateAndServe()
	}()

	log.Infof("DNS server listening at: %s", p.LocalAddr().String())
}
