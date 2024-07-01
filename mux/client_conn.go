package mux

import (
	"encoding/binary"
	"io"
	"net"
	"sync"

	E "github.com/lumavpn/luma/common/errors"
	M "github.com/lumavpn/luma/common/metadata"
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/pool"
	"github.com/lumavpn/luma/util"
)

type clientConn struct {
	net.Conn
	destination    M.Socksaddr
	requestWritten bool
	responseRead   bool
}

func (c *clientConn) NeedHandshake() bool {
	return !c.requestWritten
}

func (c *clientConn) readResponse() error {
	response, err := ReadStreamResponse(c.Conn)
	if err != nil {
		return err
	}
	if response.Status == statusError {
		return E.New("remote error: ", response.Message)
	}
	return nil
}

func (c *clientConn) Read(b []byte) (n int, err error) {
	if !c.responseRead {
		err = c.readResponse()
		if err != nil {
			return
		}
		c.responseRead = true
	}
	return c.Conn.Read(b)
}

func (c *clientConn) Write(b []byte) (n int, err error) {
	if c.requestWritten {
		return c.Conn.Write(b)
	}
	request := StreamRequest{
		Network:     "tcp",
		Destination: c.destination,
	}
	buffer := pool.NewSize(streamRequestLen(request) + len(b))
	defer buffer.Release()
	err = EncodeStreamRequest(request, buffer)
	if err != nil {
		return
	}
	buffer.Write(b)
	_, err = c.Conn.Write(buffer.Bytes())
	if err != nil {
		return
	}
	c.requestWritten = true
	return len(b), nil
}

func (c *clientConn) LocalAddr() net.Addr {
	return c.Conn.LocalAddr()
}

func (c *clientConn) RemoteAddr() net.Addr {
	return c.destination.TCPAddr()
}

func (c *clientConn) ReaderReplaceable() bool {
	return c.responseRead
}

func (c *clientConn) WriterReplaceable() bool {
	return c.requestWritten
}

func (c *clientConn) NeedAdditionalReadDeadline() bool {
	return true
}

func (c *clientConn) Upstream() any {
	return c.Conn
}

var _ N.NetPacketConn = (*clientPacketConn)(nil)

type clientPacketConn struct {
	N.AbstractConn
	conn            N.ExtendedConn
	access          sync.Mutex
	destination     M.Socksaddr
	requestWritten  bool
	responseRead    bool
	readWaitOptions N.ReadWaitOptions
}

func (c *clientPacketConn) NeedHandshake() bool {
	return !c.requestWritten
}

func (c *clientPacketConn) readResponse() error {
	response, err := ReadStreamResponse(c.conn)
	if err != nil {
		return err
	}
	if response.Status == statusError {
		return E.New("remote error: ", response.Message)
	}
	return nil
}

func (c *clientPacketConn) Read(b []byte) (n int, err error) {
	if !c.responseRead {
		err = c.readResponse()
		if err != nil {
			return
		}
		c.responseRead = true
	}
	var length uint16
	err = binary.Read(c.conn, binary.BigEndian, &length)
	if err != nil {
		return
	}
	if cap(b) < int(length) {
		return 0, io.ErrShortBuffer
	}
	return io.ReadFull(c.conn, b[:length])
}

func (c *clientPacketConn) writeRequest(payload []byte) (n int, err error) {
	request := StreamRequest{
		Network:     "udp",
		Destination: c.destination,
	}
	rLen := streamRequestLen(request)
	if len(payload) > 0 {
		rLen += 2 + len(payload)
	}
	buffer := pool.NewSize(rLen)
	defer buffer.Release()
	err = EncodeStreamRequest(request, buffer)
	if err != nil {
		return
	}
	if len(payload) > 0 {
		util.Must(
			binary.Write(buffer, binary.BigEndian, uint16(len(payload))),
			util.Error(buffer.Write(payload)),
		)
	}
	_, err = c.conn.Write(buffer.Bytes())
	if err != nil {
		return
	}
	c.requestWritten = true
	return len(payload), nil
}

func (c *clientPacketConn) Write(b []byte) (n int, err error) {
	if !c.requestWritten {
		c.access.Lock()
		if c.requestWritten {
			c.access.Unlock()
		} else {
			defer c.access.Unlock()
			return c.writeRequest(b)
		}
	}
	err = binary.Write(c.conn, binary.BigEndian, uint16(len(b)))
	if err != nil {
		return
	}
	return c.conn.Write(b)
}

func (c *clientPacketConn) ReadBuffer(buffer *pool.Buffer) (err error) {
	if !c.responseRead {
		err = c.readResponse()
		if err != nil {
			return
		}
		c.responseRead = true
	}
	var length uint16
	err = binary.Read(c.conn, binary.BigEndian, &length)
	if err != nil {
		return
	}
	_, err = buffer.ReadFullFrom(c.conn, int(length))
	return
}

func (c *clientPacketConn) WriteBuffer(buffer *pool.Buffer) error {
	if !c.requestWritten {
		c.access.Lock()
		if c.requestWritten {
			c.access.Unlock()
		} else {
			defer c.access.Unlock()
			defer buffer.Release()
			return util.Error(c.writeRequest(buffer.Bytes()))
		}
	}
	bLen := buffer.Len()
	binary.BigEndian.PutUint16(buffer.ExtendHeader(2), uint16(bLen))
	return c.conn.WriteBuffer(buffer)
}

func (c *clientPacketConn) FrontHeadroom() int {
	return 2
}

func (c *clientPacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if !c.responseRead {
		err = c.readResponse()
		if err != nil {
			return
		}
		c.responseRead = true
	}
	var length uint16
	err = binary.Read(c.conn, binary.BigEndian, &length)
	if err != nil {
		return
	}
	if cap(p) < int(length) {
		return 0, nil, io.ErrShortBuffer
	}
	n, err = io.ReadFull(c.conn, p[:length])
	return
}

func (c *clientPacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if !c.requestWritten {
		c.access.Lock()
		if c.requestWritten {
			c.access.Unlock()
		} else {
			defer c.access.Unlock()
			return c.writeRequest(p)
		}
	}
	err = binary.Write(c.conn, binary.BigEndian, uint16(len(p)))
	if err != nil {
		return
	}
	return c.conn.Write(p)
}

func (c *clientPacketConn) ReadPacket(buffer *pool.Buffer) (destination M.Socksaddr, err error) {
	err = c.ReadBuffer(buffer)
	return
}

func (c *clientPacketConn) WritePacket(buffer *pool.Buffer, destination M.Socksaddr) error {
	return c.WriteBuffer(buffer)
}

func (c *clientPacketConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *clientPacketConn) RemoteAddr() net.Addr {
	return c.destination.UDPAddr()
}

func (c *clientPacketConn) NeedAdditionalReadDeadline() bool {
	return true
}

func (c *clientPacketConn) Upstream() any {
	return c.conn
}

var _ N.NetPacketConn = (*clientPacketAddrConn)(nil)

type clientPacketAddrConn struct {
	N.AbstractConn
	conn            N.ExtendedConn
	access          sync.Mutex
	destination     M.Socksaddr
	requestWritten  bool
	responseRead    bool
	readWaitOptions N.ReadWaitOptions
}

func (c *clientPacketAddrConn) NeedHandshake() bool {
	return !c.requestWritten
}

func (c *clientPacketAddrConn) readResponse() error {
	response, err := ReadStreamResponse(c.conn)
	if err != nil {
		return err
	}
	if response.Status == statusError {
		return E.New("remote error: ", response.Message)
	}
	return nil
}

func (c *clientPacketAddrConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if !c.responseRead {
		err = c.readResponse()
		if err != nil {
			return
		}
		c.responseRead = true
	}
	destination, err := M.SocksaddrSerializer.ReadAddrPort(c.conn)
	if err != nil {
		return
	}
	if destination.IsFqdn() {
		addr = destination
	} else {
		addr = destination.UDPAddr()
	}
	var length uint16
	err = binary.Read(c.conn, binary.BigEndian, &length)
	if err != nil {
		return
	}
	if cap(p) < int(length) {
		return 0, nil, io.ErrShortBuffer
	}
	n, err = io.ReadFull(c.conn, p[:length])
	return
}

func (c *clientPacketAddrConn) writeRequest(payload []byte, destination M.Socksaddr) (n int, err error) {
	request := StreamRequest{
		Network:     N.NetworkUDP,
		Destination: c.destination,
		PacketAddr:  true,
	}
	rLen := streamRequestLen(request)
	if len(payload) > 0 {
		rLen += M.SocksaddrSerializer.AddrPortLen(destination) + 2 + len(payload)
	}
	buffer := pool.NewSize(rLen)
	defer buffer.Release()
	err = EncodeStreamRequest(request, buffer)
	if err != nil {
		return
	}
	if len(payload) > 0 {
		err = M.SocksaddrSerializer.WriteAddrPort(buffer, destination)
		if err != nil {
			return
		}
		util.Must(
			binary.Write(buffer, binary.BigEndian, uint16(len(payload))),
			util.Error(buffer.Write(payload)),
		)
	}
	_, err = c.conn.Write(buffer.Bytes())
	if err != nil {
		return
	}
	c.requestWritten = true
	return len(payload), nil
}

func (c *clientPacketAddrConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if !c.requestWritten {
		c.access.Lock()
		if c.requestWritten {
			c.access.Unlock()
		} else {
			defer c.access.Unlock()
			return c.writeRequest(p, M.ParseSocksAddrFromNet(addr))
		}
	}
	err = M.SocksaddrSerializer.WriteAddrPort(c.conn, M.ParseSocksAddrFromNet(addr))
	if err != nil {
		return
	}
	err = binary.Write(c.conn, binary.BigEndian, uint16(len(p)))
	if err != nil {
		return
	}
	return c.conn.Write(p)
}

func (c *clientPacketAddrConn) ReadPacket(buffer *pool.Buffer) (destination M.Socksaddr, err error) {
	if !c.responseRead {
		err = c.readResponse()
		if err != nil {
			return
		}
		c.responseRead = true
	}
	destination, err = M.SocksaddrSerializer.ReadAddrPort(c.conn)
	if err != nil {
		return
	}
	var length uint16
	err = binary.Read(c.conn, binary.BigEndian, &length)
	if err != nil {
		return
	}
	_, err = buffer.ReadFullFrom(c.conn, int(length))
	return
}

func (c *clientPacketAddrConn) WritePacket(buffer *pool.Buffer, destination M.Socksaddr) error {
	if !c.requestWritten {
		c.access.Lock()
		if c.requestWritten {
			c.access.Unlock()
		} else {
			defer c.access.Unlock()
			defer buffer.Release()
			return util.Error(c.writeRequest(buffer.Bytes(), destination))
		}
	}
	bLen := buffer.Len()
	header := pool.With(buffer.ExtendHeader(M.SocksaddrSerializer.AddrPortLen(destination) + 2))
	err := M.SocksaddrSerializer.WriteAddrPort(header, destination)
	if err != nil {
		return err
	}
	util.Must(binary.Write(header, binary.BigEndian, uint16(bLen)))
	return c.conn.WriteBuffer(buffer)
}

func (c *clientPacketAddrConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *clientPacketAddrConn) FrontHeadroom() int {
	return 2 + M.MaxSocksaddrLength
}

func (c *clientPacketAddrConn) NeedAdditionalReadDeadline() bool {
	return true
}

func (c *clientPacketAddrConn) Upstream() any {
	return c.conn
}
