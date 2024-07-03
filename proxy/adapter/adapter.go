package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"time"

	C "github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/common/atomic"
	"github.com/lumavpn/luma/common/queue"
	"github.com/lumavpn/luma/dialer"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy"
	"github.com/lumavpn/luma/util"

	"github.com/puzpuzpuz/xsync/v3"
)

var UnifiedDelay = atomic.NewBool(false)

const (
	defaultHistoriesNum = 10
)

type internalProxyState struct {
	alive   atomic.Bool
	history *queue.Queue[proxy.DelayHistory]
}

type Proxy struct {
	proxy.ProxyAdapter
	alive   atomic.Bool
	history *queue.Queue[proxy.DelayHistory]
	extra   *xsync.MapOf[string, *internalProxyState]
}

// AliveForTestUrl implements C.Proxy
func (p *Proxy) AliveForTestUrl(url string) bool {
	if state, ok := p.extra.Load(url); ok {
		return state.alive.Load()
	}

	return p.alive.Load()
}

// Dial implements C.Proxy
func (p *Proxy) Dial(metadata *M.Metadata) (proxy.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), C.DefaultTCPTimeout)
	defer cancel()
	return p.DialContext(ctx, metadata)
}

// DialContext implements C.ProxyAdapter
func (p *Proxy) DialContext(ctx context.Context, metadata *M.Metadata, opts ...dialer.Option) (proxy.Conn, error) {
	conn, err := p.ProxyAdapter.DialContext(ctx, metadata, opts...)
	return conn, err
}

// DialUDP implements C.ProxyAdapter
func (p *Proxy) DialUDP(metadata *M.Metadata) (proxy.PacketConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), C.DefaultUDPTimeout)
	defer cancel()
	return p.ListenPacketContext(ctx, metadata)
}

// ListenPacketContext implements C.ProxyAdapter
func (p *Proxy) ListenPacketContext(ctx context.Context, metadata *M.Metadata, opts ...dialer.Option) (proxy.PacketConn, error) {
	pc, err := p.ProxyAdapter.ListenPacketContext(ctx, metadata, opts...)
	return pc, err
}

// DelayHistory implements C.Proxy
func (p *Proxy) DelayHistory() []proxy.DelayHistory {
	queueM := p.history.Copy()
	histories := []proxy.DelayHistory{}
	for _, item := range queueM {
		histories = append(histories, item)
	}
	return histories
}

// DelayHistoryForTestUrl implements C.Proxy
func (p *Proxy) DelayHistoryForTestUrl(url string) []proxy.DelayHistory {
	var queueM []proxy.DelayHistory

	if state, ok := p.extra.Load(url); ok {
		queueM = state.history.Copy()
	}
	histories := []proxy.DelayHistory{}
	for _, item := range queueM {
		histories = append(histories, item)
	}
	return histories
}

// ExtraDelayHistories return all delay histories for each test URL
// implements C.Proxy
func (p *Proxy) ExtraDelayHistories() map[string]proxy.ProxyState {
	histories := map[string]proxy.ProxyState{}

	p.extra.Range(func(k string, v *internalProxyState) bool {
		testUrl := k
		state := v

		queueM := state.history.Copy()
		var history []proxy.DelayHistory

		for _, item := range queueM {
			history = append(history, item)
		}

		histories[testUrl] = proxy.ProxyState{
			Alive:   state.alive.Load(),
			History: history,
		}
		return true
	})
	return histories
}

// LastDelayForTestUrl return last history record of the specified URL. if proxy is not alive, return the max value of uint16.
// implements C.Proxy
func (p *Proxy) LastDelayForTestUrl(url string) (delay uint16) {
	var maxDelay uint16 = 0xffff

	alive := false
	var history proxy.DelayHistory

	if state, ok := p.extra.Load(url); ok {
		alive = state.alive.Load()
		history = state.history.Last()
	}

	if !alive || history.Delay == 0 {
		return maxDelay
	}
	return history.Delay
}

// MarshalJSON implements C.ProxyAdapter
func (p *Proxy) MarshalJSON() ([]byte, error) {
	inner, err := p.ProxyAdapter.MarshalJSON()
	if err != nil {
		return inner, err
	}

	mapping := map[string]any{}
	_ = json.Unmarshal(inner, &mapping)
	mapping["history"] = p.DelayHistory()
	mapping["extra"] = p.ExtraDelayHistories()
	mapping["alive"] = p.alive.Load()
	mapping["name"] = p.Name()
	mapping["udp"] = p.SupportUDP()
	mapping["xudp"] = p.SupportXUDP()
	mapping["tfo"] = p.SupportTFO()
	return json.Marshal(mapping)
}

// URLTest get the delay for the specified URL
// implements C.Proxy
func (p *Proxy) URLTest(ctx context.Context, url string, expectedStatus util.IntRanges[uint16]) (t uint16, err error) {
	var satisfied bool

	defer func() {
		alive := err == nil
		record := proxy.DelayHistory{Time: time.Now()}
		if alive {
			record.Delay = t
		}

		p.alive.Store(alive)
		p.history.Put(record)
		if p.history.Len() > defaultHistoriesNum {
			p.history.Pop()
		}

		state, ok := p.extra.Load(url)
		if !ok {
			state = &internalProxyState{
				history: queue.New[proxy.DelayHistory](defaultHistoriesNum),
				alive:   atomic.NewBool(true),
			}
			p.extra.Store(url, state)
		}

		if !satisfied {
			record.Delay = 0
			alive = false
		}

		state.alive.Store(alive)
		state.history.Put(record)
		if state.history.Len() > defaultHistoriesNum {
			state.history.Pop()
		}

	}()

	unifiedDelay := UnifiedDelay.Load()

	addr, err := urlToMetadata(url)
	if err != nil {
		return
	}

	start := time.Now()
	instance, err := p.DialContext(ctx, &addr)
	if err != nil {
		return
	}
	defer func() {
		_ = instance.Close()
	}()

	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return
	}
	req = req.WithContext(ctx)

	transport := &http.Transport{
		DialContext: func(context.Context, string, string) (net.Conn, error) {
			return instance, nil
		},
		// from http.DefaultTransport
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	defer client.CloseIdleConnections()

	resp, err := client.Do(req)

	if err != nil {
		return
	}

	_ = resp.Body.Close()

	if unifiedDelay {
		second := time.Now()
		resp, err = client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			start = second
		}
	}

	satisfied = resp != nil && (expectedStatus == nil || expectedStatus.Check(uint16(resp.StatusCode)))
	t = uint16(time.Since(start) / time.Millisecond)
	return
}
func NewProxy(adapter proxy.ProxyAdapter) *Proxy {
	return &Proxy{
		ProxyAdapter: adapter,
		history:      queue.New[proxy.DelayHistory](defaultHistoriesNum),
		alive:        atomic.NewBool(true),
		extra:        xsync.NewMapOf[string, *internalProxyState]()}
}

func urlToMetadata(rawURL string) (addr M.Metadata, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return
	}

	port := u.Port()
	if port == "" {
		switch u.Scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		default:
			err = fmt.Errorf("%s scheme not Support", rawURL)
			return
		}
	}
	uintPort, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return
	}

	addr = M.Metadata{
		Host:    u.Hostname(),
		DstIP:   netip.Addr{},
		DstPort: uint16(uintPort),
	}
	return
}
