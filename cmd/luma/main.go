package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/automaxprocs/maxprocs"
)

func main() {
	_, _ = maxprocs.Set(maxprocs.Logger(func(string, ...any) {}))

	fmt.Println("Starting new instance")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
}
