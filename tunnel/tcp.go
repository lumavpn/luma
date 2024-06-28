package tunnel

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
)

const (
	// tcpWaitTimeout implements a TCP half-close timeout.
	tcpWaitTimeout = 60 * time.Second

	defaultTCPTimeout = 5 * time.Second
)

func (t *tunnel) handleTCPConn(c adapter.TCPConn) {
	conn := c.Conn()
	defer func(c net.Conn) {
		_ = c.Close()
	}(conn)

	m := c.Metadata()
	if !m.Valid() {
		log.Debugf("[Metadata] not valid: %#v", m)
		return
	}

	preHandleFailed := false
	if err := preHandleMetadata(m); err != nil {
		log.Debugf("[Metadata PreHandle] error: %s", err)
		preHandleFailed = true
	}

	// If both trials have failed, we can do nothing but give up
	if preHandleFailed {
		log.Debugf("Metadata prehandle failed for connection %s --> %s",
			m.SourceAddress(), m.DestinationAddress())
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

	proxy := t.resolveMetadata(m)
	/*var peekBytes []byte
	var peekLen int*/
	ctx, cancel := context.WithTimeout(context.Background(), defaultTCPTimeout)
	defer cancel()

	remoteConn, err := proxy.DialContext(ctx, m)
	if err != nil {
		log.Warnf("[TCP] dial %s: %v", m.DestinationAddress(), err)
		return
	}
	m.MidIP, m.MidPort = parseAddr(remoteConn.LocalAddr())

	log.Infof("[TCP] %s <-> %s", m.SourceAddress(), m.DestinationAddress())
	_ = conn.SetReadDeadline(time.Now()) // stop unfinished peek
	peekMutex.Lock()
	defer peekMutex.Unlock()
	_ = conn.SetReadDeadline(time.Time{}) // reset
	handleSocket(conn, remoteConn)
}

func (t *tunnel) resolveMetadata(m *metadata.Metadata) proxy.Proxy {
	return proxy.NewDirect()
}
