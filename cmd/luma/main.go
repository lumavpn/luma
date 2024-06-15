package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/lumavpn/luma"
	v "github.com/lumavpn/luma/common/version"
	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/log"
	"go.uber.org/automaxprocs/maxprocs"
)

func main() {
	var configFile string
	var version bool
	flag.StringVar(&configFile, "config", "config.yaml", "YAML format configuration file")
	flag.BoolVar(&version, "v", false, "show current version of luma")

	if version {
		log.Debug(v.String())
		os.Exit(0)
	}

	_, _ = maxprocs.Set(maxprocs.Logger(func(string, ...any) {}))

	ctx := context.Background()

	cfg, err := config.Init(configFile)
	if err != nil {
		log.Fatal(err)
	}

	lu := luma.New(cfg)
	if err := lu.Start(ctx); err != nil {
		log.Fatalf("unable to create luma: %v", err)
	}
	defer lu.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
}
