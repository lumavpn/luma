package rw

import (
	"encoding/binary"
	"io"
)

func Skip(reader io.Reader) error {
	return SkipN(reader, 1)
}

func SkipN(reader io.Reader, size int) error {
	_, err := io.CopyN(Discard, reader, int64(size))
	return err
}

func ReadByte(reader io.Reader) (byte, error) {
	if br, isBr := reader.(io.ByteReader); isBr {
		return br.ReadByte()
	}
	var b [1]byte
	if _, err := io.ReadFull(reader, b[:]); err != nil {
		return 0, err
	}
	return b[0], nil
}

func ReadBytes(reader io.Reader, size int) ([]byte, error) {
	b := make([]byte, size)
	if _, err := io.ReadFull(reader, b); err != nil {
		return nil, err
	}
	return b, nil
}

func ReadString(reader io.Reader, size int) (string, error) {
	b, err := ReadBytes(reader, size)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type stubByteReader struct {
	io.Reader
}

func (r stubByteReader) ReadByte() (byte, error) {
	return ReadByte(r.Reader)
}

func ToByteReader(reader io.Reader) io.ByteReader {
	if byteReader, ok := reader.(io.ByteReader); ok {
		return byteReader
	}
	return &stubByteReader{reader}
}

func ReadVString(reader io.Reader) (string, error) {
	length, err := binary.ReadUvarint(ToByteReader(reader))
	if err != nil {
		return "", err
	}
	value, err := ReadBytes(reader, int(length))
	if err != nil {
		return "", err
	}
	return string(value), nil
}
