package tunnel

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/conn"
	"github.com/lumavpn/luma/dns/resolver"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	P "github.com/lumavpn/luma/proxy"
)

const (
	// tcpWaitTimeout implements a TCP half-close timeout.
	tcpWaitTimeout = 60 * time.Second

	defaultTCPTimeout = 5 * time.Second
)

func (t *tunnel) handleTCPConn(tcpConn adapter.TCPConn) {
	defer func(c net.Conn) {
		_ = c.Close()
	}(tcpConn.Conn())

	m := tcpConn.Metadata()
	if !m.Valid() {
		log.Debugf("[Metadata] not valid: %#v", m)
		return
	}

	preHandleFailed := false
	if err := preHandleMetadata(m); err != nil {
		log.Debugf("[Metadata PreHandle] error: %s", err)
		preHandleFailed = true
	}

	c := tcpConn.Conn()
	c.ResetPeeked()

	// If both trials have failed, we can do nothing but give up
	if preHandleFailed {
		log.Debugf("Metadata prehandle failed for connection %s --> %s",
			m.SourceAddress(), m.DestinationAddress())
		return
	}

	peekMutex := sync.Mutex{}
	if !c.Peeked() {
		peekMutex.Lock()
		go func() {
			defer peekMutex.Unlock()
			_ = c.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			_, _ = c.Peek(1)
			_ = c.SetReadDeadline(time.Time{})
		}()
	}

	proxy := t.resolveMetadata(m)

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

	remoteConn, err := retry(ctx, func(ctx context.Context) (remoteConn P.Conn, err error) {
		remoteConn, err = proxy.DialContext(ctx, dialMetadata)
		if err != nil {
			return
		}

		if conn.NeedHandshake(remoteConn) {
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
			peekBytes, _ = c.Peek(c.Buffered())
			_, err = remoteConn.Write(peekBytes)
			if err != nil {
				return
			}
			if peekLen = len(peekBytes); peekLen > 0 {
				_, _ = c.Discard(peekLen)
			}
		}
		return
	}, func(err error) {
		if err != nil {
			log.Error(err)
		}
	})
	if err != nil {
		return
	}
	defer func(remoteConn P.Conn) {
		_ = remoteConn.Close()
	}(remoteConn)

	m.MidIP, m.MidPort = parseAddr(remoteConn.LocalAddr())

	log.Infof("[TCP] %s <-> %s", m.SourceAddress(), m.DestinationAddress())
	_ = c.SetReadDeadline(time.Now()) // stop unfinished peek
	peekMutex.Lock()
	defer peekMutex.Unlock()
	_ = c.SetReadDeadline(time.Time{}) // reset
	handleSocket(c, remoteConn)
}

func (t *tunnel) resolveMetadata(m *metadata.Metadata) proxy.Proxy {
	return proxy.NewDirect()
}
