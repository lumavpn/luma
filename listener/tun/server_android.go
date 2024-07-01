package tun

import (
	"runtime"

	"github.com/breakfreesoftware/netlink"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/stack"
	"golang.org/x/sys/unix"
)

func (l *Listener) buildAndroidRules(tunOptions *stack.Options) error {
	packageManager, err := stack.NewPackageManager(l.handler)
	if err != nil {
		return err
	}
	err = packageManager.Start()
	if err != nil {
		return err
	}
	l.packageManager = packageManager
	tunOptions.BuildAndroidRules(packageManager, l.handler)
	return nil
}

func (h *ListenerHandler) OnPackagesUpdated(packages int, sharedUsers int) {
	return
}

func (l *Listener) openAndroidHotspot(tunOptions stack.Options) {
	if runtime.GOOS == "android" && tunOptions.AutoRoute {
		priority := 9000
		if len(tunOptions.ExcludedRanges()) > 0 {
			priority++
		}
		if tunOptions.InterfaceMonitor.AndroidVPNEnabled() {
			priority++
		}
		it := netlink.NewRule()
		it.Priority = priority
		it.IifName = tunOptions.Name
		it.Table = 254 //main
		it.Family = unix.AF_INET
		it.SuppressPrefixlen = 0
		err := netlink.RuleAdd(it)
		if err != nil {
			log.Warn("[TUN] add AndroidHotspot rule error")
		}
	}
}
