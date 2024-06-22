package tunnel

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/common/network"
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

	peekMutex := sync.Mutex{}
	if !conn.Peeked() {
		peekMutex.Lock()
		go func() {
			defer peekMutex.Unlock()
			_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			_, _ = conn.Peek(1)
			_ = conn.SetReadDeadline(time.Time{})
		}()
	}

	proxy, rule, err := t.resolveMetadata(metadata)
	if err != nil {
		log.Warnf("[Metadata] parse failed: %s", err.Error())
		return
	}

	var peekBytes []byte
	var peekLen int
	ctx, cancel := context.WithTimeout(context.Background(), common.DefaultTCPTimeout)
	defer cancel()
	remoteConn, err := retry(ctx, func(ctx context.Context) (remoteConn C.Conn, err error) {
		remoteConn, err = proxy.DialContext(ctx, metadata)
		if err != nil {
			return
		}
		if network.NeedHandshake(remoteConn) {
			defer func() {
				for _, chain := range remoteConn.Chains() {
					if chain == "REJECT" {
						err = nil
						return
					}
				}
				if err != nil {
					remoteConn = nil
				}
			}()
			peekMutex.Lock()
			defer peekMutex.Unlock()
			peekBytes, _ = conn.Peek(conn.Buffered())
			_, err = remoteConn.Write(peekBytes)
			if err != nil {
				return
			}
			if peekLen = len(peekBytes); peekLen > 0 {
				_, _ = conn.Discard(peekLen)
			}
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
	_ = conn.SetReadDeadline(time.Now()) // stop unfinished peek
	peekMutex.Lock()
	defer peekMutex.Unlock()
	_ = conn.SetReadDeadline(time.Time{}) // reset
	network.HandleSocket(conn, remoteConn)
}
