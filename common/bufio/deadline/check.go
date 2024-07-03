package deadline

import (
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/util"
)

type WithoutReadDeadline interface {
	NeedAdditionalReadDeadline() bool
}

func NeedAdditionalReadDeadline(rawReader any) bool {
	if deadlineReader, loaded := rawReader.(WithoutReadDeadline); loaded {
		return deadlineReader.NeedAdditionalReadDeadline()
	}
	if upstream, hasUpstream := rawReader.(N.WithUpstreamReader); hasUpstream {
		return NeedAdditionalReadDeadline(upstream.UpstreamReader())
	}
	if upstream, hasUpstream := rawReader.(util.WithUpstream); hasUpstream {
		return NeedAdditionalReadDeadline(upstream.Upstream())
	}
	return false
}
