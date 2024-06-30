package adapter

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/miekg/dns"
)

type DNSType string

const (
	DNSTypeHost   DNSType = "host"
	DNSTypeFakeIP DNSType = "fakeip"
	DNSTypeRaw    DNSType = "raw"
)

type DNSContext struct {
	context.Context

	id  uuid.UUID
	msg *dns.Msg
	tp  string
}

func NewDNSContext(ctx context.Context, msg *dns.Msg) *DNSContext {
	id, _ := uuid.NewV4()
	return &DNSContext{
		Context: ctx,
		id:      id,
		msg:     msg,
	}
}

func (c *DNSContext) ID() uuid.UUID {
	return c.id
}

func (c *DNSContext) SetType(tp string) {
	c.tp = tp
}

func (c *DNSContext) Type() string {
	return c.tp
}
