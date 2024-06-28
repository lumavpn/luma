package luma

import (
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
