package luma

import (
	"net"
	"net/netip"
	"runtime"
	"strconv"

	"github.com/lumavpn/luma/log"
)

func checkTunName(tunName string) (ok bool) {
	defer func() {
		if !ok {
			log.Warnf("[TUN] Unsupported tunName(%s) in %s, force regenerate by ourselves.", tunName, runtime.GOOS)
		}
	}()
	log.Debugf("tun name is %s", tunName)
	if runtime.GOOS == "darwin" {
		if len(tunName) <= 4 {
			return false
		}
		if tunName[:4] != "utun" {
			return false
		}
		if _, parseErr := strconv.ParseInt(tunName[4:], 10, 16); parseErr != nil {
			return false
		}
	}
	return true
}

func verifyIP6() bool {
	if iAddrs, err := net.InterfaceAddrs(); err == nil {
		for _, addr := range iAddrs {
			if prefix, err := netip.ParsePrefix(addr.String()); err == nil {
				if addr := prefix.Addr().Unmap(); addr.Is6() && addr.IsGlobalUnicast() {
					return true
				}
			}
		}
	}
	return false
}
