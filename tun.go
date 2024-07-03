package luma

import (
	"context"
	"slices"
	"sort"

	"github.com/lumavpn/luma/component/ebpf"
	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/listener/tun"
	"github.com/lumavpn/luma/log"
)

func (lu *Luma) recreateRedirToTun(ifaceNames []string, tunDevice string) error {
	lu.tcMux.Lock()
	defer lu.tcMux.Unlock()

	nicArr := ifaceNames
	slices.Sort(nicArr)
	nicArr = slices.Compact(nicArr)

	log.Debugf("recreateRedirToTun. ifaceNames %v tunDevice %s", ifaceNames, tunDevice)

	if lu.tcProgram != nil {
		lu.tcProgram.Close()
		lu.tcProgram = nil
	}

	if len(nicArr) == 0 {
		return nil
	}

	program, err := ebpf.NewTcEBpfProgram(nicArr, tunDevice)
	if err != nil {
		log.Errorf("Attached tc ebpf program error: %v", err)
		return err
	}
	lu.tcProgram = program

	log.Infof("Attached tc ebpf program to interfaces %v", lu.tcProgram.RawNICs())
	return nil
}

func (lu *Luma) startTunListener(ctx context.Context, cfg *config.Tun) error {
	lu.tunMu.Lock()
	tunConfig := cfg
	defer func() {
		lu.lastTunConfig = *tunConfig
		lu.tunMu.Unlock()
	}()
	var err error
	defer func() {
		if err != nil {
			log.Errorf("Start TUN listening error: %s", err.Error())
			tunConfig.Enable = false
		}
	}()

	if !lu.hasTunConfigChange(tunConfig) {
		if lu.tunListener != nil {
			lu.tunListener.FlushDefaultInterface()
		}
		return nil
	}

	lu.closeTunListener()

	if !tunConfig.Enable {
		return nil
	}

	log.Debugf("Device name is %s", tunConfig.Device)

	listener, err := tun.New(tunConfig, lu.tunnel)
	if err != nil {
		return err
	}
	lu.setTunListener(listener)

	log.Infof("[TUN] Tun adapter listening at: %s", listener.Address())
	return nil
}

func (lu *Luma) setTunListener(l *tun.Listener) {
	lu.mu.Lock()
	defer lu.mu.Unlock()
	lu.tunListener = l
}

func (lu *Luma) closeTunListener() {
	if lu.tunListener != nil {
		lu.tunListener.Close()
		lu.tunListener = nil
	}
}

func (lu *Luma) hasTunConfigChange(tunConfig *config.Tun) bool {
	if lu.lastTunConfig.Enable != tunConfig.Enable ||
		lu.lastTunConfig.Device != tunConfig.Device ||
		lu.lastTunConfig.Stack != tunConfig.Stack ||
		lu.lastTunConfig.AutoRoute != tunConfig.AutoRoute ||
		lu.lastTunConfig.AutoDetectInterface != tunConfig.AutoDetectInterface ||
		lu.lastTunConfig.MTU != tunConfig.MTU ||
		lu.lastTunConfig.StrictRoute != tunConfig.StrictRoute ||
		lu.lastTunConfig.EndpointIndependentNat != tunConfig.EndpointIndependentNat ||
		lu.lastTunConfig.UDPTimeout != tunConfig.UDPTimeout ||
		lu.lastTunConfig.FileDescriptor != tunConfig.FileDescriptor {
		return true
	}

	if len(lu.lastTunConfig.DNSHijack) != len(tunConfig.DNSHijack) {
		return true
	}

	sort.Slice(tunConfig.DNSHijack, func(i, j int) bool {
		return tunConfig.DNSHijack[i] < tunConfig.DNSHijack[j]
	})

	sort.Slice(tunConfig.Inet4Address, func(i, j int) bool {
		return tunConfig.Inet4Address[i].String() < tunConfig.Inet4Address[j].String()
	})

	sort.Slice(tunConfig.Inet6Address, func(i, j int) bool {
		return tunConfig.Inet6Address[i].String() < tunConfig.Inet6Address[j].String()
	})

	sort.Slice(tunConfig.Inet4RouteAddress, func(i, j int) bool {
		return tunConfig.Inet4RouteAddress[i].String() < tunConfig.Inet4RouteAddress[j].String()
	})

	sort.Slice(tunConfig.Inet6RouteAddress, func(i, j int) bool {
		return tunConfig.Inet6RouteAddress[i].String() < tunConfig.Inet6RouteAddress[j].String()
	})

	sort.Slice(tunConfig.Inet4RouteExcludeAddress, func(i, j int) bool {
		return tunConfig.Inet4RouteExcludeAddress[i].String() < tunConfig.Inet4RouteExcludeAddress[j].String()
	})

	sort.Slice(tunConfig.Inet6RouteExcludeAddress, func(i, j int) bool {
		return tunConfig.Inet6RouteExcludeAddress[i].String() < tunConfig.Inet6RouteExcludeAddress[j].String()
	})

	sort.Slice(tunConfig.IncludeUID, func(i, j int) bool {
		return tunConfig.IncludeUID[i] < tunConfig.IncludeUID[j]
	})

	sort.Slice(tunConfig.IncludeUIDRange, func(i, j int) bool {
		return tunConfig.IncludeUIDRange[i] < tunConfig.IncludeUIDRange[j]
	})

	sort.Slice(tunConfig.ExcludeUID, func(i, j int) bool {
		return tunConfig.ExcludeUID[i] < tunConfig.ExcludeUID[j]
	})

	sort.Slice(tunConfig.ExcludeUIDRange, func(i, j int) bool {
		return tunConfig.ExcludeUIDRange[i] < tunConfig.ExcludeUIDRange[j]
	})

	sort.Slice(tunConfig.IncludeAndroidUser, func(i, j int) bool {
		return tunConfig.IncludeAndroidUser[i] < tunConfig.IncludeAndroidUser[j]
	})

	sort.Slice(tunConfig.IncludePackage, func(i, j int) bool {
		return tunConfig.IncludePackage[i] < tunConfig.IncludePackage[j]
	})

	sort.Slice(tunConfig.ExcludePackage, func(i, j int) bool {
		return tunConfig.ExcludePackage[i] < tunConfig.ExcludePackage[j]
	})

	if !slices.Equal(tunConfig.DNSHijack, lu.lastTunConfig.DNSHijack) ||
		!slices.Equal(tunConfig.Inet4Address, lu.lastTunConfig.Inet4Address) ||
		!slices.Equal(tunConfig.Inet6Address, lu.lastTunConfig.Inet6Address) ||
		!slices.Equal(tunConfig.Inet4RouteAddress, lu.lastTunConfig.Inet4RouteAddress) ||
		!slices.Equal(tunConfig.Inet6RouteAddress, lu.lastTunConfig.Inet6RouteAddress) ||
		!slices.Equal(tunConfig.Inet4RouteExcludeAddress, lu.lastTunConfig.Inet4RouteExcludeAddress) ||
		!slices.Equal(tunConfig.Inet6RouteExcludeAddress, lu.lastTunConfig.Inet6RouteExcludeAddress) ||
		!slices.Equal(tunConfig.IncludeUID, lu.lastTunConfig.IncludeUID) ||
		!slices.Equal(tunConfig.IncludeUIDRange, lu.lastTunConfig.IncludeUIDRange) ||
		!slices.Equal(tunConfig.ExcludeUID, lu.lastTunConfig.ExcludeUID) ||
		!slices.Equal(tunConfig.ExcludeUIDRange, lu.lastTunConfig.ExcludeUIDRange) ||
		!slices.Equal(tunConfig.IncludeAndroidUser, lu.lastTunConfig.IncludeAndroidUser) ||
		!slices.Equal(tunConfig.IncludePackage, lu.lastTunConfig.IncludePackage) ||
		!slices.Equal(tunConfig.ExcludePackage, lu.lastTunConfig.ExcludePackage) {
		return true
	}

	return false
}
