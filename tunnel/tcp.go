package tunnel

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/dns/resolver"
	"github.com/lumavpn/luma/log"
	proxyAdapter "github.com/lumavpn/luma/proxy/adapter"
)

const (
	// tcpWaitTimeout implements a TCP half-close timeout.
	tcpWaitTimeout = 60 * time.Second

	defaultTCPTimeout = 5 * time.Second
)

func (t *tunnel) handleTCPConn(c adapter.TCPConn) {
	defer func(c net.Conn) {
		_ = c.Close()
	}(c.Conn())

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

	conn := c.Conn()
	conn.ResetPeeked()

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

	proxy, rule, err := t.resolveMetadata(m)
	if err != nil {
		log.Warnf("[Metadata] parse failed: %s", err.Error())
		return
	}

	dialMetadata := m
	if len(m.Host) > 0 {
		if node, ok := resolver.DefaultHosts.Search(m.Host, false); ok {
			if dstIp, _ := node.RandIP(); !t.fakeIPRange.Contains(dstIp) {
				dialMetadata.DstIP = dstIp
				dialMetadata.DNSMode = common.DNSHosts
				dialMetadata = dialMetadata.Pure()
			}
		}
	}

	var peekBytes []byte
	var peekLen int
	ctx, cancel := context.WithTimeout(context.Background(), defaultTCPTimeout)
	defer cancel()

	remoteConn, err := retry(ctx, func(ctx context.Context) (remoteConn proxyAdapter.Conn, err error) {
		remoteConn, err = proxy.DialContext(ctx, dialMetadata)
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
		return
	}, func(err error) {
		if rule == nil {
			log.Warnf(
				"[TCP] dial %s %s --> %s error: %s",
				proxy.Name(),
				m.SourceAddress(),
				m.DestinationAddress(),
				err.Error(),
			)
		} else {
			log.Warnf("[TCP] dial %s (match %s/%s) %s --> %s error: %s", proxy.Name(), rule.Rule().String(),
				rule.Payload(), m.SourceAddress(), m.DestinationAddress(), err.Error())
		}
	})
	if err != nil {
		return
	}
	//log.Infof("[TCP] %s <-> %s", m.SourceAddress(), m.DestinationAddress())
	_ = conn.SetReadDeadline(time.Now()) // stop unfinished peek
	peekMutex.Lock()
	defer peekMutex.Unlock()
	_ = conn.SetReadDeadline(time.Time{}) // reset

	handleSocket(conn, remoteConn)
}
