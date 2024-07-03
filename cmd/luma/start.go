//go:build !with_notray

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/lumavpn/systray"
	luma "github.com/lumavpn/luma"
	"github.com/lumavpn/luma/log"
)

func Start(ctx context.Context, lu *luma.Luma) {
	st := NewSysTray(lu)

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM,
			syscall.SIGQUIT)

		select {
		case sig := <-c:
			log.Infof("got signal to exit: %v", sig)
			systray.Quit()
		case <-st.closing:
			log.Info("luma exiting...")
		}
	}()

	systray.Run(st.TrayReady(ctx), st.Exit) // system tray management
}
