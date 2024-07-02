package mux

import (
	"errors"
	"net"
	"sync"

	"github.com/sagernet/sing/common/buf"
	M "github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/common/network"
)

type packet struct {
	conn  *network.NetPacketConn
	mutex *sync.Mutex
	rAddr net.Addr
	lAddr net.Addr
	buff  *buf.Buffer
}

func (c *packet) Data() []byte {
	return c.buff.Bytes()
}

// WriteBack wirtes UDP packet with source(ip, port) = `addr`
func (c *packet) WriteBack(b []byte, addr net.Addr) (n int, err error) {
	if addr == nil {
		err = errors.New("address is invalid")
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	conn := *c.conn
	if conn == nil {
		err = errors.New("writeBack to closed connection")
		return
	}

	buff := buf.NewPacket()
	defer buff.Release()
	n, err = buff.Write(b)
	if err != nil {
		return
	}

	err = conn.WritePacket(buff, M.SocksaddrFromNet(addr))
	if err != nil {
		return
	}
	return
}

// LocalAddr returns the source IP/Port of UDP Packet
func (c *packet) LocalAddr() net.Addr {
	return c.rAddr
}

func (c *packet) Drop() {
	c.buff.Release()
}

func (c *packet) InAddr() net.Addr {
	return c.lAddr
}

func ShouldIgnorePacketError(err error) bool {
	/*// ignore simple error
	if errors.IsTimeout(err) || errors.IsClosed(err) || errors.IsCanceled(err) {
		return true
	}*/
	return false
}
