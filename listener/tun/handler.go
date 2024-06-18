package tun

import (
	"context"
	"net"

	M "github.com/lumavpn/luma/common/metadata"
)

func (h *ListenerHandler) NewConnection(ctx context.Context, conn net.Conn, metadata M.Metadata) error {
	return nil
}

func (h *ListenerHandler) NewPacketConnection(ctx context.Context, conn net.PacketConn, metadata M.Metadata) error {
	return nil
}

func NewListenerHandler(lc ListenerConfig) (h *ListenerHandler, err error) {
	h = &ListenerHandler{ListenerConfig: lc}
	return
}
