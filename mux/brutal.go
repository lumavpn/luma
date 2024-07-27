package mux

import (
	"encoding/binary"
	"io"

	E "github.com/lumavpn/luma/common/errors"
	"github.com/lumavpn/luma/common/rw"
	"github.com/lumavpn/luma/util"
	"github.com/sagernet/sing/common/buf"
)

const (
	BrutalExchangeDomain = "_BrutalBwExchange"
	BrutalMinSpeedBPS    = 65536
)

func WriteBrutalRequest(writer io.Writer, receiveBPS uint64) error {
	return binary.Write(writer, binary.BigEndian, receiveBPS)
}

func ReadBrutalRequest(reader io.Reader) (uint64, error) {
	var receiveBPS uint64
	err := binary.Read(reader, binary.BigEndian, &receiveBPS)
	return receiveBPS, err
}

func WriteBrutalResponse(writer io.Writer, receiveBPS uint64, ok bool, message string) error {
	buffer := buf.New()
	defer buffer.Release()
	util.Must(binary.Write(buffer, binary.BigEndian, ok))
	if ok {
		util.Must(binary.Write(buffer, binary.BigEndian, receiveBPS))
	} else {
		err := rw.WriteVString(buffer, message)
		if err != nil {
			return err
		}
	}
	return util.Error(writer.Write(buffer.Bytes()))
}

func ReadBrutalResponse(reader io.Reader) (uint64, error) {
	var ok bool
	err := binary.Read(reader, binary.BigEndian, &ok)
	if err != nil {
		return 0, err
	}
	if ok {
		var receiveBPS uint64
		err = binary.Read(reader, binary.BigEndian, &receiveBPS)
		return receiveBPS, err
	} else {
		var message string
		message, err = rw.ReadVString(reader)
		if err != nil {
			return 0, err
		}
		return 0, E.New("remote error: ", message)
	}
}
