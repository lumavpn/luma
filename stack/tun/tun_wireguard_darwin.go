//go:build with_gvisor && darwin

package tun

import (
	"fmt"
	"sync"

	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/stack/device/iobased"

	"golang.zx2c4.com/wireguard/tun"
)

type TUN struct {
	*iobased.Endpoint

	nt     *tun.NativeTun
	mtu    uint32
	name   string
	offset int

	rSizes []int
	rBuffs [][]byte
	wBuffs [][]byte
	rMutex sync.Mutex
	wMutex sync.Mutex
}

func New(options Options) (Device, error) {
	t := &TUN{
		name:   options.Name,
		mtu:    options.MTU,
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
	t.nt = nt.(*tun.NativeTun)
	ep, err := iobased.New(t, t.mtu, offset)
	if err != nil {
		return nil, fmt.Errorf("create endpoint: %w", err)
	}
	t.Endpoint = ep

	return t, nil
}

func (t *TUN) Read(packet []byte) (int, error) {
	t.rMutex.Lock()
	defer t.rMutex.Unlock()
	t.rBuffs[0] = packet
	_, err := t.nt.Read(t.rBuffs, t.rSizes, t.offset)
	return t.rSizes[0], err
}

func (t *TUN) Write(packet []byte) (int, error) {
	t.wMutex.Lock()
	defer t.wMutex.Unlock()
	t.wBuffs[0] = packet
	return t.nt.Write(t.wBuffs, t.offset)
}

func (t *TUN) Name() string {
	name, _ := t.nt.Name()
	return name
}

func (t *TUN) Close() {
	defer t.Endpoint.Close()
	t.nt.Close()
}
