package stack

import (
	"fmt"
	"net/netip"
	"runtime"

	"go4.org/netipx"
)

const autoRouteUseSubRanges = runtime.GOOS == "darwin"

func (o *Options) BuildAutoRouteRanges(underNetworkExtension bool) ([]netip.Prefix, error) {
	var routeRanges []netip.Prefix
	if o.AutoRoute && len(o.Inet4Address) > 0 {
		var inet4Ranges []netip.Prefix
		if len(o.Inet4RouteAddress) > 0 {
			inet4Ranges = o.Inet4RouteAddress
		} else if autoRouteUseSubRanges && !underNetworkExtension {
			inet4Ranges = []netip.Prefix{
				netip.PrefixFrom(netip.AddrFrom4([4]byte{0: 1}), 8),
				netip.PrefixFrom(netip.AddrFrom4([4]byte{0: 2}), 7),
				netip.PrefixFrom(netip.AddrFrom4([4]byte{0: 4}), 6),
				netip.PrefixFrom(netip.AddrFrom4([4]byte{0: 8}), 5),
				netip.PrefixFrom(netip.AddrFrom4([4]byte{0: 16}), 4),
				netip.PrefixFrom(netip.AddrFrom4([4]byte{0: 32}), 3),
				netip.PrefixFrom(netip.AddrFrom4([4]byte{0: 64}), 2),
				netip.PrefixFrom(netip.AddrFrom4([4]byte{0: 128}), 1),
			}
		} else {
			inet4Ranges = []netip.Prefix{netip.PrefixFrom(netip.IPv4Unspecified(), 0)}
		}
		if len(o.Inet4RouteExcludeAddress) == 0 {
			routeRanges = append(routeRanges, inet4Ranges...)
		} else {
			var builder netipx.IPSetBuilder
			for _, inet4Range := range inet4Ranges {
				builder.AddPrefix(inet4Range)
			}
			for _, prefix := range o.Inet4RouteExcludeAddress {
				builder.RemovePrefix(prefix)
			}
			resultSet, err := builder.IPSet()
			if err != nil {
				return nil, fmt.Errorf("build IPv4 route address: %v", err)
			}
			routeRanges = append(routeRanges, resultSet.Prefixes()...)
		}
	}
	if len(o.Inet6Address) > 0 {
		var inet6Ranges []netip.Prefix
		if len(o.Inet6RouteAddress) > 0 {
			inet6Ranges = o.Inet6RouteAddress
		} else if autoRouteUseSubRanges && !underNetworkExtension {
			inet6Ranges = []netip.Prefix{
				netip.PrefixFrom(netip.AddrFrom16([16]byte{0: 1}), 8),
				netip.PrefixFrom(netip.AddrFrom16([16]byte{0: 2}), 7),
				netip.PrefixFrom(netip.AddrFrom16([16]byte{0: 4}), 6),
				netip.PrefixFrom(netip.AddrFrom16([16]byte{0: 8}), 5),
				netip.PrefixFrom(netip.AddrFrom16([16]byte{0: 16}), 4),
				netip.PrefixFrom(netip.AddrFrom16([16]byte{0: 32}), 3),
				netip.PrefixFrom(netip.AddrFrom16([16]byte{0: 64}), 2),
				netip.PrefixFrom(netip.AddrFrom16([16]byte{0: 128}), 1),
			}
		} else {
			inet6Ranges = []netip.Prefix{netip.PrefixFrom(netip.IPv6Unspecified(), 0)}
		}
		if len(o.Inet6RouteExcludeAddress) == 0 {
			routeRanges = append(routeRanges, inet6Ranges...)
		} else {
			var builder netipx.IPSetBuilder
			for _, inet6Range := range inet6Ranges {
				builder.AddPrefix(inet6Range)
			}
			for _, prefix := range o.Inet6RouteExcludeAddress {
				builder.RemovePrefix(prefix)
			}
			resultSet, err := builder.IPSet()
			if err != nil {
				return nil, fmt.Errorf("build IPv6 route address: %v", err)
			}
			routeRanges = append(routeRanges, resultSet.Prefixes()...)
		}
	}
	return routeRanges, nil
}
