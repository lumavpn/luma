package util

import "net"

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

func Top(obj any) any {
	if u, ok := obj.(WithUpstream); ok {
		return Top(u.Upstream())
	}
	if u, ok := obj.(stdWithUpstreamNetConn); ok {
		return Top(u.NetConn())
	}
	return obj
}
