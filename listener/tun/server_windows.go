package tun

import (
	"time"

	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/stack/tun"
)

func tunNew(options tun.Options) (tunIf tun.Tun, err error) {
	maxRetry := 3
	for i := 0; i < maxRetry; i++ {
		timeBegin := time.Now()
		tunIf, err = tun.New(options)
		if err == nil {
			return
		}
		timeEnd := time.Now()
		if timeEnd.Sub(timeBegin) < 1*time.Second { // retrying for "Cannot create a file when that file already exists."
			return
		}
		log.Warnf("Start Tun interface timeout: %s [retrying %d/%d]", err, i+1, maxRetry)
	}
	return
}

func init() {
	//stack.TunnelType = InterfaceName
}