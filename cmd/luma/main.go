package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/lumavpn/luma/log"
	"go.uber.org/automaxprocs/maxprocs"
)

func main() {
	_, _ = maxprocs.Set(maxprocs.Logger(func(string, ...any) {}))

	log.Debug("Starting new instance")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
}
