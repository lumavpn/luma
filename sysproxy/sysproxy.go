package sysproxy

import (
	"strconv"

	"github.com/lumavpn/luma/log"
)

const (
	// Names used when configuring system proxy
	webproxy       = "webproxy"
	securewebproxy = "securewebproxy"
	socksproxy     = "socksproxy"
)

type Opts struct {
	EnableSecureWebProxy bool
	EnableWebProxy       bool
	EnableSocksProxy     bool
	Host                 string
	Port                 string
}

type systemProxyHandler struct {
	*Opts
}

func EnableAll(port int) error {
	return Enable(&Opts{
		EnableSecureWebProxy: true,
		EnableWebProxy:       true,
		EnableSocksProxy:     true,
		Host:                 "127.0.0.1",
		Port:                 strconv.Itoa(port),
	})
}

func Enable(opts *Opts) error {
	log.Debugf("Enabling system proxy on port %v", opts.Port)
	return EnableSystemProxy(&Opts{
		EnableSecureWebProxy: opts.EnableSecureWebProxy,
		EnableWebProxy:       opts.EnableWebProxy,
		EnableSocksProxy:     opts.EnableSocksProxy,
		Host:                 "127.0.0.1",
		Port:                 opts.Port,
	})
}

func Disable() {
	DisableSystemProxy()
}
