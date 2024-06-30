package dns

import (
	"net/netip"

	"github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/common/cache"
	"github.com/lumavpn/luma/component/fakeip"
)

type ResolverEnhancer struct {
	mode     common.DNSMode
	fakePool *fakeip.Pool
	mapping  *cache.LruCache[netip.Addr, string]
}

func (h *ResolverEnhancer) FakeIPEnabled() bool {
	return h.mode == common.DNSFakeIP
}

func NewEnhancer(opts *Options) *ResolverEnhancer {
	var fakePool *fakeip.Pool
	var mapping *cache.LruCache[netip.Addr, string]

	if opts.EnhancedMode != common.DNSNormal {
		fakePool = opts.Pool
		mapping = cache.New(cache.WithSize[netip.Addr, string](4096))
	}

	return &ResolverEnhancer{
		mode:     opts.EnhancedMode,
		fakePool: fakePool,
		mapping:  mapping,
	}
}
