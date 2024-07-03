package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/lumavpn/luma"
	"github.com/lumavpn/luma/config"
	v "github.com/lumavpn/luma/internal/version"
	"github.com/lumavpn/luma/log"
	"go.uber.org/automaxprocs/maxprocs"
)

var (
	cmdConfig  = config.New()
	configFile string
	homeDir    string
	version    bool
)

func init() {
	flag.StringVar(&configFile, "config", "config.yaml", "YAML format configuration file")
	flag.StringVar(&homeDir, "d", "", "set configuration directory")
	flag.BoolVar(&cmdConfig.EnableTun2socks, "enable-tun2socks", false, "enable tun2socks. default: false")
	flag.IntVar(&cmdConfig.Mark, "fwmark", 0, "Set firewall MARK (Linux only)")
	flag.IntVar(&cmdConfig.MTU, "mtu", 0, "Set device maximum transmission unit (MTU)")
	flag.StringVar(&cmdConfig.LogLevel, "loglevel", "info", "Log level [debug|info|warning|error|silent]")
	flag.StringVar(&cmdConfig.RawTun.Device, "device", "", "Use this device [driver://]name")
	flag.StringVar(&cmdConfig.RawTun.Interface, "interface", "", "Use network INTERFACE (Linux/MacOS only)")
	flag.BoolVar(&version, "v", false, "show current version of luma")
	flag.Parse()
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	maxprocs.Set(maxprocs.Logger(func(string, ...any) {}))

	if version {
		log.Debug(v.String())
		os.Exit(0)
	}

	ctx := context.Background()

	cfg := config.Init(configFile, cmdConfig)
	lu, err := luma.New(cfg)
	checkErr(err)
	Start(ctx, lu)

	defer lu.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
}
