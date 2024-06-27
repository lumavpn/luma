package tun

import (
	"net/netip"
	"runtime"
)

const (
	androidUserRange        = 100000
	userEnd          uint32 = 0xFFFFFFFF - 1
)

const autoRouteUseSubRanges = runtime.GOOS == "darwin"

func (o *Options) BuildAutoRouteRanges(underNetworkExtension bool) ([]netip.Prefix, error) {
	var routeRanges []netip.Prefix
	return routeRanges, nil
}
