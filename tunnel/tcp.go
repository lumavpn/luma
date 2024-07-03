package tunnel

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/common"
	C "github.com/lumavpn/luma/common"
	N "github.com/lumavpn/luma/common/net"
	"github.com/lumavpn/luma/component/resolver"
	"github.com/lumavpn/luma/log"
	P "github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/tunnel/sniffer"
	"github.com/lumavpn/luma/tunnel/statistic"
)

func (t *tunnel) handleTCPConn(connCtx adapter.TCPConn) {

	if !t.isHandle(connCtx.Metadata().Type) {
		_ = connCtx.Conn().Close()
		return
	}

	defer func(conn net.Conn) {
		_ = conn.Close()
	}(connCtx.Conn())

	m := connCtx.Metadata()
	if !m.Valid() {
		log.Debugf("[Metadata] not valid: %#v", m)
		return
	}

	preHandleFailed := false
	if err := preHandleMetadata(m); err != nil {
		log.Debugf("[Metadata PreHandle] error: %s", err)
		preHandleFailed = true
	}

	conn := connCtx.Conn()
	conn.ResetPeeked() // reset before sniffer
	if sniffer.Dispatcher.Enable() && t.sniffingEnable {
		// Try to sniff a domain when `preHandleMetadata` failed, this is usually
		// caused by a "Fake DNS record missing" error when enhanced-mode is fake-ip.
		if sniffer.Dispatcher.TCPSniff(conn, m) {
			// we now have a domain name
			preHandleFailed = false
		}
	}

	// If both trials have failed, we can do nothing but give up
	if preHandleFailed {
		log.Debugf("[Metadata PreHandle] failed to sniff a domain for connection %s --> %s, give up",
			m.SourceDetail(), m.RemoteAddress())
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

	proxy, rule, err := t.proxyDialer.ResolveMetadata(m)
	if err != nil {
		log.Warnf("[Metadata] parse failed: %s", err.Error())
		return
	}
	dialMetadata := m
	if len(m.Host) > 0 {
		if node, ok := resolver.DefaultHosts.Search(m.Host, false); ok {
			if dstIp, _ := node.RandIP(); !t.fakeIPRange.Contains(dstIp) {
				dialMetadata.DstIP = dstIp
				dialMetadata.DNSMode = C.DNSHosts
				dialMetadata = dialMetadata.Pure()
			}
		}
	}
	var peekBytes []byte
	var peekLen int

	ctx, cancel := context.WithTimeout(context.Background(), common.DefaultTCPTimeout)
	defer cancel()
	remoteConn, err := retry(ctx, func(ctx context.Context) (remoteConn P.Conn, err error) {
		remoteConn, err = proxy.DialContext(ctx, dialMetadata)
		if err != nil {
			return
		}

		if N.NeedHandshake(remoteConn) {
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
				m.SourceDetail(),
				m.RemoteAddress(),
				err.Error(),
			)
		} else {
			//log.Warnf("[TCP] dial %s (match %s/%s) %s --> %s error: %s", proxy.Name(), rule.RuleType().String(), rule.Payload(), metadata.SourceDetail(), metadata.RemoteAddress(), err.Error())
		}
	})
	if err != nil {
		return
	}
	remoteConn = statistic.NewTCPTracker(remoteConn, statistic.DefaultManager, m, rule, 0, int64(peekLen), true)
	defer func(remoteConn P.Conn) {
		_ = remoteConn.Close()
	}(remoteConn)
	mode := t.mode
	switch true {
	case m.SpecialProxy != "":
		log.Infof("[TCP] %s --> %s using %s", m.SourceDetail(), m.RemoteAddress(), m.SpecialProxy)
	case rule != nil:
		if rule.Payload() != "" {
			log.Infof("[TCP] %s --> %s match %s using %s", m.SourceDetail(), m.RemoteAddress(), fmt.Sprintf("%s(%s)", rule.RuleType().String(), rule.Payload()), remoteConn.Chains().String())
		} else {
			log.Infof("[TCP] %s --> %s match %s using %s", m.SourceDetail(), m.RemoteAddress(), rule.RuleType().String(), remoteConn.Chains().String())
		}
	case mode == C.Global:
		log.Infof("[TCP] %s --> %s using GLOBAL", m.SourceDetail(), m.RemoteAddress())
	case mode == C.Direct:
		log.Infof("[TCP] %s --> %s using DIRECT", m.SourceDetail(), m.RemoteAddress())
	case mode == C.Select && proxy != nil:
		log.Infof("[TCP] %s --> %s using PROXY %s", m.SourceDetail(), m.RemoteAddress(), proxy.Addr())
	default:
		log.Infof(
			"[TCP] %s --> %s doesn't match any rule using DIRECT",
			m.SourceDetail(),
			m.RemoteAddress(),
		)
	}

	_ = conn.SetReadDeadline(time.Now()) // stop unfinished peek
	peekMutex.Lock()
	defer peekMutex.Unlock()
	_ = conn.SetReadDeadline(time.Time{}) // reset
	handleSocket(conn, remoteConn)

}
