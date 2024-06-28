package util

import (
	"io"
	"net"
)

func DefaultValue[T any]() T {
	var defaultValue T
	return defaultValue
}

func Error(_ any, err error) error {
	return err
}

func Filter[T any](arr []T, block func(it T) bool) []T {
	var retArr []T
	for _, it := range arr {
		if block(it) {
			retArr = append(retArr, it)
		}
	}
	return retArr
}

func All[T any](array []T, block func(it T) bool) bool {
	for _, it := range array {
		if !block(it) {
			return false
		}
	}
	return true
}

func FilterNotNil[T any](arr []T) []T {
	return Filter(arr, func(it T) bool {
		var anyIt any = it
		return anyIt != nil
	})
}

func FlatMap[T any, N any](arr []T, block func(it T) []N) []N {
	var retAddr []N
	for _, item := range arr {
		retAddr = append(retAddr, block(item)...)
	}
	return retAddr
}

func Map[T any, N any](arr []T, block func(it T) N) []N {
	retArr := make([]N, 0, len(arr))
	for index := range arr {
		retArr = append(retArr, block(arr[index]))
	}
	return retArr
}

type WithUpstream interface {
	Upstream() any
}

type stdWithUpstreamNetConn interface {
	NetConn() net.Conn
}

func Cast[T any](obj any) (T, bool) {
	if c, ok := obj.(T); ok {
		return c, true
	}
	if u, ok := obj.(WithUpstream); ok {
		return Cast[T](u.Upstream())
	}
	if u, ok := obj.(stdWithUpstreamNetConn); ok {
		return Cast[T](u.NetConn())
	}
	return DefaultValue[T](), false
}

func Close(closers ...any) error {
	var retErr error
	for _, closer := range closers {
		if closer == nil {
			continue
		}
		switch c := closer.(type) {
		case io.Closer:
			err := c.Close()
			if err != nil {
				retErr = err
			}
			continue
		case WithUpstream:
			err := Close(c.Upstream())
			if err != nil {
				retErr = err
			}
		}
	}
	return retErr
}
