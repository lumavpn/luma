package conn

import (
	"net"

	"github.com/gofrs/uuid/v5"
	M "github.com/lumavpn/luma/metadata"
)

type ConnContext struct {
	id       uuid.UUID
	metadata *M.Metadata
	conn     *BuffConn
}

func NewConnContext(conn net.Conn, metadata *M.Metadata) (*ConnContext, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return &ConnContext{
		id:       id,
		metadata: metadata,
		conn:     NewBuffConn(conn),
	}, nil
}

func (c *ConnContext) ID() uuid.UUID {
	return c.id
}

func (c *ConnContext) Metadata() *M.Metadata {
	return c.metadata
}

func (c *ConnContext) Conn() net.Conn {
	return c.conn
}

type PacketConnContext struct {
	id         uuid.UUID
	metadata   *M.Metadata
	packetConn net.PacketConn
}

func NewPacketConnContext(conn net.PacketConn, metadata *M.Metadata) (*PacketConnContext, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return &PacketConnContext{
		id:         id,
		metadata:   metadata,
		packetConn: conn,
	}, nil
}

func (pc *PacketConnContext) ID() uuid.UUID {
	return pc.id
}

func (pc *PacketConnContext) Metadata() *M.Metadata {
	return pc.metadata
}

func (pc *PacketConnContext) PacketConn() net.PacketConn {
	return pc.packetConn
}
