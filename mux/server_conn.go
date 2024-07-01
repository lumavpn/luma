package mux

import (
	"encoding/binary"
	"io"
	"net"
	"sync"

	M "github.com/lumavpn/luma/common/metadata"
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/pool"
	"github.com/lumavpn/luma/common/rw"
	"github.com/lumavpn/luma/util"
)

type serverConn struct {
	N.ExtendedConn
	responseWritten bool
}

func (c *serverConn) NeedHandshake() bool {
	return !c.responseWritten
}

func (c *serverConn) HandshakeFailure(err error) error {
	errMessage := err.Error()
	buffer := pool.NewSize(1 + rw.UVariantLen(uint64(len(errMessage))) + len(errMessage))
	defer buffer.Release()
	util.Must(
		buffer.WriteByte(statusError),
		rw.WriteVString(buffer, errMessage),
	)
	return util.Error(c.ExtendedConn.Write(buffer.Bytes()))
}

func (c *serverConn) Write(b []byte) (n int, err error) {
	if c.responseWritten {
		return c.ExtendedConn.Write(b)
	}
	buffer := pool.NewSize(1 + len(b))
	defer buffer.Release()
	util.Must(
		buffer.WriteByte(statusSuccess),
		util.Error(buffer.Write(b)),
	)
	_, err = c.ExtendedConn.Write(buffer.Bytes())
	if err != nil {
		return
	}
	c.responseWritten = true
	return len(b), nil
}

func (c *serverConn) WriteBuffer(buffer *pool.Buffer) error {
	if c.responseWritten {
		return c.ExtendedConn.WriteBuffer(buffer)
	}
	buffer.ExtendHeader(1)[0] = statusSuccess
	c.responseWritten = true
	return c.ExtendedConn.WriteBuffer(buffer)
}

func (c *serverConn) FrontHeadroom() int {
	if !c.responseWritten {
		return 1
	}
	return 0
}

func (c *serverConn) NeedAdditionalReadDeadline() bool {
	return true
}

func (c *serverConn) Upstream() any {
	return c.ExtendedConn
}

type serverPacketConn struct {
	N.ExtendedConn
	access          sync.Mutex
	destination     M.Socksaddr
	responseWritten bool
}

func (c *serverPacketConn) NeedHandshake() bool {
	return !c.responseWritten
}

func (c *serverPacketConn) HandshakeFailure(err error) error {
	errMessage := err.Error()
	buffer := pool.NewSize(1 + rw.UVariantLen(uint64(len(errMessage))) + len(errMessage))
	defer buffer.Release()
	util.Must(
		buffer.WriteByte(statusError),
		rw.WriteVString(buffer, errMessage),
	)
	return util.Error(c.ExtendedConn.Write(buffer.Bytes()))
}

func (c *serverPacketConn) ReadPacket(buffer *pool.Buffer) (destination M.Socksaddr, err error) {
	var length uint16
	err = binary.Read(c.ExtendedConn, binary.BigEndian, &length)
	if err != nil {
		return
	}
	_, err = buffer.ReadFullFrom(c.ExtendedConn, int(length))
	if err != nil {
		return
	}
	destination = c.destination
	return
}

func (c *serverPacketConn) WritePacket(buffer *pool.Buffer, destination M.Socksaddr) error {
	pLen := buffer.Len()
	util.Must(binary.Write(pool.With(buffer.ExtendHeader(2)), binary.BigEndian, uint16(pLen)))
	if !c.responseWritten {
		c.access.Lock()
		if c.responseWritten {
			c.access.Unlock()
		} else {
			defer c.access.Unlock()
		}
		buffer.ExtendHeader(1)[0] = statusSuccess
		c.responseWritten = true
	}
	return c.ExtendedConn.WriteBuffer(buffer)
}

func (c *serverPacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	var length uint16
	err = binary.Read(c.ExtendedConn, binary.BigEndian, &length)
	if err != nil {
		return
	}
	if cap(p) < int(length) {
		return 0, nil, io.ErrShortBuffer
	}
	n, err = io.ReadFull(c.ExtendedConn, p[:length])
	return
}

func (c *serverPacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if !c.responseWritten {
		c.access.Lock()
		if c.responseWritten {
			c.access.Unlock()
		} else {
			defer c.access.Unlock()
			_, err = c.ExtendedConn.Write([]byte{statusSuccess})
			if err != nil {
				return
			}
			c.responseWritten = true
		}
	}
	err = binary.Write(c.ExtendedConn, binary.BigEndian, uint16(len(p)))
	if err != nil {
		return
	}
	return c.ExtendedConn.Write(p)
}

func (c *serverPacketConn) NeedAdditionalReadDeadline() bool {
	return true
}

func (c *serverPacketConn) Upstream() any {
	return c.ExtendedConn
}

func (c *serverPacketConn) FrontHeadroom() int {
	if !c.responseWritten {
		return 3
	}
	return 2
}

type serverPacketAddrConn struct {
	N.ExtendedConn
	access          sync.Mutex
	responseWritten bool
}

func (c *serverPacketAddrConn) NeedHandshake() bool {
	return !c.responseWritten
}

func (c *serverPacketAddrConn) HandshakeFailure(err error) error {
	errMessage := err.Error()
	buffer := pool.NewSize(1 + rw.UVariantLen(uint64(len(errMessage))) + len(errMessage))
	defer buffer.Release()
	util.Must(
		buffer.WriteByte(statusError),
		rw.WriteVString(buffer, errMessage),
	)
	return util.Error(c.ExtendedConn.Write(buffer.Bytes()))
}

func (c *serverPacketAddrConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	destination, err := M.SocksaddrSerializer.ReadAddrPort(c.ExtendedConn)
	if err != nil {
		return
	}
	if destination.IsFqdn() {
		addr = destination
	} else {
		addr = destination.UDPAddr()
	}
	var length uint16
	err = binary.Read(c.ExtendedConn, binary.BigEndian, &length)
	if err != nil {
		return
	}
	if cap(p) < int(length) {
		return 0, nil, io.ErrShortBuffer
	}
	n, err = io.ReadFull(c.ExtendedConn, p[:length])
	return
}

func (c *serverPacketAddrConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if !c.responseWritten {
		c.access.Lock()
		if c.responseWritten {
			c.access.Unlock()
		} else {
			defer c.access.Unlock()
			_, err = c.ExtendedConn.Write([]byte{statusSuccess})
			if err != nil {
				return
			}
			c.responseWritten = true
		}
	}
	err = M.SocksaddrSerializer.WriteAddrPort(c.ExtendedConn, M.ParseSocksAddrFromNet(addr))
	if err != nil {
		return
	}
	err = binary.Write(c.ExtendedConn, binary.BigEndian, uint16(len(p)))
	if err != nil {
		return
	}
	return c.ExtendedConn.Write(p)
}

func (c *serverPacketAddrConn) ReadPacket(buffer *pool.Buffer) (destination M.Socksaddr, err error) {
	destination, err = M.SocksaddrSerializer.ReadAddrPort(c.ExtendedConn)
	if err != nil {
		return
	}
	var length uint16
	err = binary.Read(c.ExtendedConn, binary.BigEndian, &length)
	if err != nil {
		return
	}
	_, err = buffer.ReadFullFrom(c.ExtendedConn, int(length))
	if err != nil {
		return
	}
	return
}

func (c *serverPacketAddrConn) WritePacket(buffer *pool.Buffer, destination M.Socksaddr) error {
	pLen := buffer.Len()
	util.Must(binary.Write(pool.With(buffer.ExtendHeader(2)), binary.BigEndian, uint16(pLen)))
	err := M.SocksaddrSerializer.WriteAddrPort(pool.With(buffer.ExtendHeader(M.SocksaddrSerializer.AddrPortLen(destination))), destination)
	if err != nil {
		return err
	}
	if !c.responseWritten {
		c.access.Lock()
		if c.responseWritten {
			c.access.Unlock()
		} else {
			defer c.access.Unlock()
			buffer.ExtendHeader(1)[0] = statusSuccess
			c.responseWritten = true
		}
	}
	return c.ExtendedConn.WriteBuffer(buffer)
}

func (c *serverPacketAddrConn) NeedAdditionalReadDeadline() bool {
	return true
}

func (c *serverPacketAddrConn) Upstream() any {
	return c.ExtendedConn
}

func (c *serverPacketAddrConn) FrontHeadroom() int {
	if !c.responseWritten {
		return 3 + M.MaxSocksaddrLength
	}
	return 2 + M.MaxSocksaddrLength
}