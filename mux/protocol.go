package mux

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"time"

	M "github.com/lumavpn/luma/common/metadata"
	"github.com/lumavpn/luma/common/pool"
	"github.com/lumavpn/luma/common/rw"
)

const (
	ProtocolSmux = iota
	ProtocolYAMux
	ProtocolH2Mux
)

const (
	Version0 = iota
	Version1
)

const (
	TCPTimeout = 5 * time.Second
)

var Destination = M.Socksaddr{
	Fqdn: "mux",
	Port: 444,
}

type Request struct {
	Version  byte
	Protocol byte
	Padding  bool
}

func ReadRequest(reader io.Reader) (*Request, error) {
	version, err := rw.ReadByte(reader)
	if err != nil {
		return nil, err
	}
	if version < Version0 || version > Version1 {
		return nil, fmt.Errorf("unsupported version: %v", version)
	}
	protocol, err := rw.ReadByte(reader)
	if err != nil {
		return nil, err
	}
	var paddingEnabled bool
	if version == Version1 {
		err = binary.Read(reader, binary.BigEndian, &paddingEnabled)
		if err != nil {
			return nil, err
		}
		if paddingEnabled {
			var paddingLen uint16
			err = binary.Read(reader, binary.BigEndian, &paddingLen)
			if err != nil {
				return nil, err
			}
			err = rw.SkipN(reader, int(paddingLen))
			if err != nil {
				return nil, err
			}
		}
	}
	return &Request{Version: version, Protocol: protocol, Padding: paddingEnabled}, nil
}

func EncodeRequest(request Request, payload []byte) *pool.Buffer {
	var requestLen int
	requestLen += 2
	var paddingLen uint16
	if request.Version == Version1 {
		requestLen += 1
		if request.Padding {
			requestLen += 2
			paddingLen = uint16(256 + rand.Intn(512))
			requestLen += int(paddingLen)
		}
	}
	b := pool.NewSize(requestLen + len(payload))
	b.WriteByte(request.Version)
	b.WriteByte(request.Protocol)
	if request.Version == Version1 {
		binary.Write(b, binary.BigEndian, request.Padding)
		if request.Padding {
			binary.Write(b, binary.BigEndian, paddingLen)
			b.Extend(int(paddingLen))
		}
	}
	b.Write(payload)
	return b
}

const (
	flagUDP       = 1
	flagAddr      = 2
	statusSuccess = 0
	statusError   = 1
)

type StreamRequest struct {
	Network     string
	Destination M.Socksaddr
	PacketAddr  bool
}

func ReadStreamRequest(reader io.Reader) (*StreamRequest, error) {
	var flags uint16
	err := binary.Read(reader, binary.BigEndian, &flags)
	if err != nil {
		return nil, err
	}
	destination, err := M.SocksaddrSerializer.ReadAddrPort(reader)
	if err != nil {
		return nil, err
	}
	var network string
	var udpAddr bool
	if flags&flagUDP == 0 {
		network = "tcp"
	} else {
		network = "udp"
		udpAddr = flags&flagAddr != 0
	}
	return &StreamRequest{network, destination, udpAddr}, nil
}

func streamRequestLen(request StreamRequest) int {
	var rLen int
	rLen += 1 // version
	rLen += 2 // flags
	rLen += M.SocksaddrSerializer.AddrPortLen(request.Destination)
	return rLen
}

func EncodeStreamRequest(request StreamRequest, buffer *pool.Buffer) error {
	destination := request.Destination
	var flags uint16
	if request.Network == "udp" {
		flags |= flagUDP
	}
	if request.PacketAddr {
		flags |= flagAddr
		if !destination.IsValid() {
			destination = Destination
		}
	}
	binary.Write(buffer, binary.BigEndian, flags)
	return M.SocksaddrSerializer.WriteAddrPort(buffer, destination)
}

type StreamResponse struct {
	Status  uint8
	Message string
}

func ReadStreamResponse(reader io.Reader) (*StreamResponse, error) {
	var response StreamResponse
	status, err := rw.ReadByte(reader)
	if err != nil {
		return nil, err
	}
	response.Status = status
	if status == statusError {
		response.Message, err = rw.ReadVString(reader)
		if err != nil {
			return nil, err
		}
	}
	return &response, nil
}
