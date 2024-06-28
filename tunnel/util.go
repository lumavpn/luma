package tunnel

import (
	"io"
	"net"
	"net/netip"
	"time"

	"github.com/lumavpn/luma/metadata"
)

type ReadOnlyReader struct {
	io.Reader
}

type WriteOnlyWriter struct {
	io.Writer
}

func preHandleMetadata(m *metadata.Metadata) error {
	// handle IP string on host
	if ip, err := netip.ParseAddr(m.Host); err == nil {
		m.DstIP = ip
		m.Host = ""
	}
	return nil
}

// handleSocket copies between left and right bidirectionally.
func handleSocket(leftConn, rightConn net.Conn) {
	ch := make(chan error)

	go func() {
		_, err := io.Copy(WriteOnlyWriter{Writer: leftConn}, ReadOnlyReader{Reader: rightConn})
		leftConn.SetReadDeadline(time.Now())
		ch <- err
	}()

	io.Copy(WriteOnlyWriter{Writer: rightConn}, ReadOnlyReader{Reader: leftConn})
	rightConn.SetReadDeadline(time.Now())
	<-ch
}
