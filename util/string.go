package util

import (
	"fmt"
	"io"
	"regexp"
	"strconv"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func StringTitle(s string) string {
	return cases.Title(language.English, cases.Compact).String(s)
}

func ReverseString(s string) string {
	a := []rune(s)
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}
	return string(a)
}

var rateStringRegexp = regexp.MustCompile(`^(\d+)\s*([KMGT]?)([Bb])ps$`)

func StringToBps(s string) uint64 {
	if s == "" {
		return 0
	}

	// when have not unit, use Mbps
	if v, err := strconv.Atoi(s); err == nil {
		return StringToBps(fmt.Sprintf("%d Mbps", v))
	}

	m := rateStringRegexp.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	var n uint64 = 1
	switch m[2] {
	case "T":
		n *= 1000
		fallthrough
	case "G":
		n *= 1000
		fallthrough
	case "M":
		n *= 1000
		fallthrough
	case "K":
		n *= 1000
	}
	v, _ := strconv.ParseUint(m[1], 10, 64)
	n *= v
	if m[3] == "b" {
		// Bits, need to convert to bytes
		n /= 8
	}
	return n
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
