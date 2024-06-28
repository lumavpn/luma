//go:build with_wireguard && with_gvisor && darwin

package tun

import (
	"fmt"
	"sync"

	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/stack/device/iobased"
	"github.com/sagernet/sing/common/buf"
	"gvisor.dev/gvisor/pkg/tcpip/stack"

	"golang.zx2c4.com/wireguard/tun"
)

var _ GVisorTun = (*NativeTun)(nil)

func (t *NativeTun) NewEndpoint() (stack.LinkEndpoint, error) {
	e := &DarwinEndpoint{
		name:   t.name,
		mtu:    t.mtu,
		offset: offset,
		rSizes: make([]int, 1),
		rBuffs: make([][]byte, 1),
		wBuffs: make([][]byte, 1),
	}
	forcedMTU := defaultMTU
	if t.mtu > 0 {
		forcedMTU = int(t.mtu)
	}
	log.Debug("Opening wireguard tun..")
	nt, err := createTUN(t.name, forcedMTU)
	if err != nil {
		return nil, fmt.Errorf("create tun: %w", err)
	}
	e.tun = nt.(*tun.NativeTun)
	ep, err := iobased.New(t, t.mtu, offset)
	if err != nil {
		return nil, fmt.Errorf("create endpoint: %w", err)
	}
	e.Endpoint = ep

	return e, nil
}

var _ stack.LinkEndpoint = (*DarwinEndpoint)(nil)

type DarwinEndpoint struct {
	*iobased.Endpoint

	tun    *tun.NativeTun
	mtu    uint32
	name   string
	offset int

	rSizes []int
	rBuffs [][]byte
	wBuffs [][]byte
	rMutex sync.Mutex
	wMutex sync.Mutex
}

func (t *DarwinEndpoint) Read(packet []byte) (int, error) {
	t.rMutex.Lock()
	defer t.rMutex.Unlock()
	t.rBuffs[0] = packet
	_, err := t.tun.Read(t.rBuffs, t.rSizes, t.offset)
	return t.rSizes[0], err
}

func (t *DarwinEndpoint) Write(packet []byte) (int, error) {
	t.wMutex.Lock()
	defer t.wMutex.Unlock()
	t.wBuffs[0] = packet
	return t.tun.Write(t.wBuffs, t.offset)
}

func (t *DarwinEndpoint) Name() string {
	name, _ := t.tun.Name()
	return name
}

func (t *DarwinEndpoint) WriteVectorised(buffers []*buf.Buffer) error {
	return nil
}

func (t *DarwinEndpoint) Close() {
	defer t.Endpoint.Close()
	t.tun.Close()
}
