package util

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func Title(s string) string {
	return cases.Title(language.English, cases.Compact).String(s)
}

func ToString(messages ...any) string {
	var output string
	for _, rawMessage := range messages {
		if rawMessage == nil {
			output += "nil"
			continue
		}
		switch message := rawMessage.(type) {
		case string:
			output += message
		case bool:
			if message {
				output += "true"
			} else {
				output += "false"
			}
		case uint:
			output += strconv.FormatUint(uint64(message), 10)
		case uint8:
			output += strconv.FormatUint(uint64(message), 10)
		case uint16:
			output += strconv.FormatUint(uint64(message), 10)
		case uint32:
			output += strconv.FormatUint(uint64(message), 10)
		case uint64:
			output += strconv.FormatUint(message, 10)
		case int:
			output += strconv.FormatInt(int64(message), 10)
		case int8:
			output += strconv.FormatInt(int64(message), 10)
		case int16:
			output += strconv.FormatInt(int64(message), 10)
		case int32:
			output += strconv.FormatInt(int64(message), 10)
		case int64:
			output += strconv.FormatInt(message, 10)
		case uintptr:
			output += strconv.FormatUint(uint64(message), 10)
		case error:
			output += message.Error()
		default:
			panic("unknown value")
		}
	}
	return output
}

var rateStringRegexp = regexp.MustCompile(`^(\d+)\s*([KMGT]?)([Bb])ps$`)

func StringToBps(s string) uint64 {
	if s == "" {
		return 0
	}
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
		n /= 8
	}
	return n
}

func ReverseString(s string) string {
	a := []rune(s)
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}
	return string(a)
}

func ToStringSlice(value any) ([]string, error) {
	strArr := make([]string, 0)
	switch reflect.TypeOf(value).Kind() {
	case reflect.Slice, reflect.Array:
		origin := reflect.ValueOf(value)
		for i := 0; i < origin.Len(); i++ {
			item := fmt.Sprintf("%v", origin.Index(i))
			strArr = append(strArr, item)
		}
	case reflect.String:
		strArr = append(strArr, fmt.Sprintf("%v", value))
	default:
		return nil, errors.New("value format error, must be string or array")
	}
	return strArr, nil
}
