package util

import (
	"context"
	"io"
	"net"

	"golang.org/x/exp/constraints"
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

func Must(errs ...error) {
	for _, err := range errs {
		if err != nil {
			panic(err)
		}
	}
}

func Must1[T any](result T, err error) T {
	if err != nil {
		panic(err)
	}
	return result
}

func MinBy[T any, C constraints.Ordered](arr []T, block func(it T) C) T {
	var min T
	var minValue C
	if len(arr) == 0 {
		return min
	}
	min = arr[0]
	minValue = block(min)
	for i := 1; i < len(arr); i++ {
		item := arr[i]
		value := block(item)
		if value < minValue {
			min = item
			minValue = value
		}
	}
	return min
}

func Done(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func FilterIsInstance[T any, N any](arr []T, block func(it T) (N, bool)) []N {
	var retArr []N
	for _, it := range arr {
		if n, isN := block(it); isN {
			retArr = append(retArr, n)
		}
	}
	return retArr
}
