package tunnel

import (
	"context"
	"net"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/common"
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/log"
	C "github.com/lumavpn/luma/proxy"
)

func (t *tunnel) handleTCPConn(c adapter.TCPConn) {
	if !t.isHandle(c.Metadata().Type) {
		_ = c.Conn().Close()
		return
	}
	conn := c.Conn()
	defer func(c net.Conn) {
		_ = c.Close()
	}(conn)
	metadata := c.Metadata()
	if !metadata.Valid() {
		log.Debugf("[Metadata] not valid: %#v", metadata)
		return
	}

	preHandleFailed := false
	if err := preHandleMetadata(metadata); err != nil {
		log.Debugf("[Metadata PreHandle] error: %s", err)
		preHandleFailed = true
	}

	// If both trials have failed, we can do nothing but give up
	if preHandleFailed {
		log.Debugf("Metadata prehandle failed for connection %s --> %s",
			metadata.SourceDetail(), metadata.DestinationAddress())
		return
	}

	proxy, rule, err := t.resolveMetadata(metadata)
	if err != nil {
		log.Warnf("[Metadata] parse failed: %s", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), common.DefaultTCPTimeout)
	defer cancel()
	remoteConn, err := retry(ctx, func(ctx context.Context) (remoteConn C.Conn, err error) {
		remoteConn, err = proxy.DialContext(ctx, metadata)
		if err != nil {
			return
		}
		return remoteConn, nil
	}, func(err error) {
		if rule == nil {
			log.Warnf(
				"[TCP] dial %s %s --> %s error: %s",
				proxy.Name(),
				metadata.SourceDetail(),
				metadata.DestinationAddress(),
				err.Error(),
			)
		} else {
			log.Warnf("[TCP] dial %s (match %s/%s) %s --> %s error: %s", proxy.Name(), rule.Rule().String(),
				rule.Payload(), metadata.SourceDetail(), metadata.DestinationAddress(), err.Error())
		}
	})
	if err != nil {
		return
	}
	N.HandleSocket(conn, remoteConn)
}
