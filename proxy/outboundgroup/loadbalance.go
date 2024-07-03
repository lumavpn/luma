package outboundgroup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	lru "github.com/lumavpn/luma/common/cache"
	"github.com/lumavpn/luma/common/callback"
	N "github.com/lumavpn/luma/common/net"
	"github.com/lumavpn/luma/dialer"
	M "github.com/lumavpn/luma/metadata"
	C "github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/proxy/outbound"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/proxy/provider"
	"github.com/lumavpn/luma/proxydialer"
	"github.com/lumavpn/luma/util"
	"golang.org/x/net/publicsuffix"
)

type strategyFn = func(proxies []C.Proxy, metadata *M.Metadata, touch bool) C.Proxy

type LoadBalance struct {
	*GroupBase
	disableUDP     bool
	strategyFn     strategyFn
	testUrl        string
	expectedStatus string
	Hidden         bool
	Icon           string
}

var errStrategy = errors.New("unsupported strategy")

func parseStrategy(config map[string]any) string {
	if strategy, ok := config["strategy"].(string); ok {
		return strategy
	}
	return "consistent-hashing"
}

func getKey(metadata *M.Metadata) string {
	if metadata == nil {
		return ""
	}

	if metadata.Host != "" {
		// ip host
		if ip := net.ParseIP(metadata.Host); ip != nil {
			return metadata.Host
		}

		if etld, err := publicsuffix.EffectiveTLDPlusOne(metadata.Host); err == nil {
			return etld
		}
	}

	if !metadata.DstIP.IsValid() {
		return ""
	}

	return metadata.DstIP.String()
}

func getKeyWithSrcAndDst(metadata *M.Metadata) string {
	dst := getKey(metadata)
	src := ""
	if metadata != nil {
		src = metadata.SrcIP.String()
	}

	return fmt.Sprintf("%s%s", src, dst)
}

func jumpHash(key uint64, buckets int32) int32 {
	var b, j int64

	for j < int64(buckets) {
		b = j
		key = key*2862933555777941757 + 1
		j = int64(float64(b+1) * (float64(int64(1)<<31) / float64((key>>33)+1)))
	}

	return int32(b)
}

// DialContext implements C.ProxyAdapter
func (lb *LoadBalance) DialContext(ctx context.Context, metadata *M.Metadata, opts ...dialer.Option) (c C.Conn, err error) {
	proxy := lb.Unwrap(metadata, true)
	c, err = proxy.DialContext(ctx, metadata, lb.Base.DialOptions(opts...)...)

	if err == nil {
		c.AppendToChains(lb)
	} else {
		lb.onDialFailed(proxy.Proto(), err)
	}

	if N.NeedHandshake(c) {
		c = callback.NewFirstWriteCallBackConn(c, func(err error) {
			if err == nil {
				lb.onDialSuccess()
			} else {
				lb.onDialFailed(proxy.Proto(), err)
			}
		})
	}

	return
}

// ListenPacketContext implements C.ProxyAdapter
func (lb *LoadBalance) ListenPacketContext(ctx context.Context, metadata *M.Metadata, opts ...dialer.Option) (pc C.PacketConn, err error) {
	defer func() {
		if err == nil {
			pc.AppendToChains(lb)
		}
	}()

	proxy := lb.Unwrap(metadata, true)
	return proxy.ListenPacketContext(ctx, metadata, lb.Base.DialOptions(opts...)...)
}

// SupportUDP implements C.ProxyAdapter
func (lb *LoadBalance) SupportUDP() bool {
	return !lb.disableUDP
}

// IsL3Protocol implements C.ProxyAdapter
func (lb *LoadBalance) IsL3Protocol(metadata *M.Metadata) bool {
	return lb.Unwrap(metadata, false).IsL3Protocol(metadata)
}

func strategyRoundRobin(url string) strategyFn {
	idx := 0
	idxMutex := sync.Mutex{}
	return func(proxies []C.Proxy, metadata *M.Metadata, touch bool) C.Proxy {
		idxMutex.Lock()
		defer idxMutex.Unlock()

		i := 0
		length := len(proxies)

		if touch {
			defer func() {
				idx = (idx + i) % length
			}()
		}

		for ; i < length; i++ {
			id := (idx + i) % length
			proxy := proxies[id]
			if proxy.AliveForTestUrl(url) {
				i++
				return proxy
			}
		}

		return proxies[0]
	}
}

func strategyConsistentHashing(url string) strategyFn {
	maxRetry := 5
	return func(proxies []C.Proxy, metadata *M.Metadata, touch bool) C.Proxy {
		key := util.MapHash(getKey(metadata))
		buckets := int32(len(proxies))
		for i := 0; i < maxRetry; i, key = i+1, key+1 {
			idx := jumpHash(key, buckets)
			proxy := proxies[idx]
			if proxy.AliveForTestUrl(url) {
				return proxy
			}
		}

		// when availability is poor, traverse the entire list to get the available nodes
		for _, proxy := range proxies {
			if proxy.AliveForTestUrl(url) {
				return proxy
			}
		}

		return proxies[0]
	}
}

func strategyStickySessions(url string) strategyFn {
	ttl := time.Minute * 10
	maxRetry := 5
	lruCache := lru.New[uint64, int](
		lru.WithAge[uint64, int](int64(ttl.Seconds())),
		lru.WithSize[uint64, int](1000))
	return func(proxies []C.Proxy, metadata *M.Metadata, touch bool) C.Proxy {
		key := util.MapHash(getKeyWithSrcAndDst(metadata))
		length := len(proxies)
		idx, has := lruCache.Load(key)
		if !has {
			idx = int(jumpHash(key+uint64(time.Now().UnixNano()), int32(length)))
		}

		nowIdx := idx
		for i := 1; i < maxRetry; i++ {
			proxy := proxies[nowIdx]
			if proxy.AliveForTestUrl(url) {
				if nowIdx != idx {
					lruCache.Delete(key)
					lruCache.Set(key, nowIdx)
				}

				return proxy
			} else {
				nowIdx = int(jumpHash(key+uint64(time.Now().UnixNano()), int32(length)))
			}
		}

		lruCache.Delete(key)
		lruCache.Set(key, 0)
		return proxies[0]
	}
}

// Unwrap implements C.ProxyAdapter
func (lb *LoadBalance) Unwrap(metadata *M.Metadata, touch bool) C.Proxy {
	proxies := lb.GetProxies(touch)
	return lb.strategyFn(proxies, metadata, touch)
}

// MarshalJSON implements C.ProxyAdapter
func (lb *LoadBalance) MarshalJSON() ([]byte, error) {
	var all []string
	for _, proxy := range lb.GetProxies(false) {
		all = append(all, proxy.Name())
	}
	return json.Marshal(map[string]any{
		"type":           lb.Proto().String(),
		"all":            all,
		"testUrl":        lb.testUrl,
		"expectedStatus": lb.expectedStatus,
		"hidden":         lb.Hidden,
		"icon":           lb.Icon,
	})
}

func NewLoadBalance(option *GroupCommonOption, proxyDialer proxydialer.ProxyDialer, providers []provider.ProxyProvider, strategy string) (lb *LoadBalance, err error) {
	var strategyFn strategyFn
	switch strategy {
	case "consistent-hashing":
		strategyFn = strategyConsistentHashing(option.URL)
	case "round-robin":
		strategyFn = strategyRoundRobin(option.URL)
	case "sticky-sessions":
		strategyFn = strategyStickySessions(option.URL)
	default:
		return nil, fmt.Errorf("%w: %s", errStrategy, strategy)
	}
	return &LoadBalance{
		GroupBase: NewGroupBase(GroupBaseOption{
			outbound.BaseOption{
				Name:        option.Name,
				Proto:       proto.Proto_LoadBalance,
				Interface:   option.Interface,
				RoutingMark: option.RoutingMark,
			},
			option.Filter,
			option.ExcludeFilter,
			option.ExcludeType,
			option.TestTimeout,
			option.MaxFailedTimes,
			providers,
		}, proxyDialer),
		strategyFn:     strategyFn,
		disableUDP:     option.DisableUDP,
		testUrl:        option.URL,
		expectedStatus: option.ExpectedStatus,
		Hidden:         option.Hidden,
		Icon:           option.Icon,
	}, nil
}
