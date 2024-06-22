package outbound

import (
	"bytes"
	"encoding/base64"
	"net"
	"time"

	"github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/transport/socks5"
)

const (
	tcpKeepAlivePeriod = 30 * time.Second
)

// setKeepAlive sets the TCP keepalive option for a TCP connection
func setKeepAlive(c net.Conn) {
	if tcp, ok := c.(*net.TCPConn); ok {
		tcp.SetKeepAlive(true)
		tcp.SetKeepAlivePeriod(tcpKeepAlivePeriod)
	}
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// safeConnClose closes tcp connection safely.
func safeConnClose(c net.Conn, err error) {
	if c != nil && err != nil {
		c.Close()
	}
}

func serializesSocksAddr(metadata *metadata.Metadata) []byte {
	var buf [][]byte
	addrType := metadata.AddrType()
	aType := uint8(addrType)
	p := uint(metadata.DstPort)
	port := []byte{uint8(p >> 8), uint8(p & 0xff)}
	switch addrType {
	case socks5.AtypDomainName:
		lenM := uint8(len(metadata.Host))
		host := []byte(metadata.Host)
		buf = [][]byte{{aType, lenM}, host, port}
	case socks5.AtypIPv4:
		host := metadata.DstIP.AsSlice()
		buf = [][]byte{{aType}, host, port}
	case socks5.AtypIPv6:
		host := metadata.DstIP.AsSlice()
		buf = [][]byte{{aType}, host, port}
	}
	return bytes.Join(buf, nil)
}
