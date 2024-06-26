package tunnel

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/common/pool"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
)

const (
	// tcpWaitTimeout implements a TCP half-close timeout.
	tcpWaitTimeout = 60 * time.Second
)

func (t *tunnel) handleTCPConn(c adapter.TCPConn) {
	originConn := c.Conn()
	defer originConn.Close()

	id := c.ID()
	m := &metadata.Metadata{
		Network: metadata.TCP,
		SrcIP:   net.IP(id.RemoteAddress.AsSlice()),
		SrcPort: id.RemotePort,
		DstIP:   net.IP(id.LocalAddress.AsSlice()),
		DstPort: id.LocalPort,
	}

	remoteConn, err := proxy.Dial(m)
	if err != nil {
		log.Warnf("[TCP] dial %s: %v", m.DestinationAddress(), err)
		return
	}
	m.MidIP, m.MidPort = parseAddr(remoteConn.LocalAddr())

	log.Infof("[TCP] %s <-> %s", m.SourceAddress(), m.DestinationAddress())
	pipe(originConn, remoteConn)
}

// pipe copies copy data to & from provided net.Conn(s) bidirectionally.
func pipe(origin, remote net.Conn) {
	wg := sync.WaitGroup{}
	wg.Add(2)

	go unidirectionalStream(remote, origin, "origin->remote", &wg)
	go unidirectionalStream(origin, remote, "remote->origin", &wg)

	wg.Wait()
}

func unidirectionalStream(dst, src net.Conn, dir string, wg *sync.WaitGroup) {
	defer wg.Done()
	buf := pool.Get(pool.RelayBufferSize)
	if _, err := io.CopyBuffer(dst, src, buf); err != nil {
		log.Debugf("[TCP] copy data for %s: %v", dir, err)
	}
	pool.Put(buf)
	// Do the upload/download side TCP half-close.
	if cr, ok := src.(interface{ CloseRead() error }); ok {
		cr.CloseRead()
	}
	if cw, ok := dst.(interface{ CloseWrite() error }); ok {
		cw.CloseWrite()
	}
	// Set TCP half-close timeout.
	dst.SetReadDeadline(time.Now().Add(tcpWaitTimeout))
}
