package rw

import (
	"io"

	"github.com/lumavpn/luma/util"
)

var ZeroBytes = make([]byte, 1024)

func WriteByte(writer io.Writer, b byte) error {
	return util.Error(writer.Write([]byte{b}))
}

func WriteBytes(writer io.Writer, b []byte) error {
	return util.Error(writer.Write(b))
}

func WriteZero(writer io.Writer) error {
	return WriteByte(writer, 0)
}

func WriteZeroN(writer io.Writer, size int) error {
	var index int
	for index < size {
		next := index + 1024
		if next > size {
			_, err := writer.Write(ZeroBytes[:size-index])
			return err
		} else {
			_, err := writer.Write(ZeroBytes)
			if err != nil {
				return err
			}
			index = next
		}
	}
	return nil
}

func WriteString(writer io.Writer, str string) error {
	return WriteBytes(writer, []byte(str))
}
