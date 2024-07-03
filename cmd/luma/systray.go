//go:build !with_notray

package main

import (
	"context"
	"os"
	"sync"

	"github.com/breakfreesoftware/systray"
	luma "github.com/lumavpn/luma"
	"github.com/lumavpn/luma/icon"
	"github.com/lumavpn/luma/log"
)

type SysTray struct {
	lu      *luma.Luma
	closing chan struct{}
	mu      *sync.RWMutex
}

func NewSysTray(lu *luma.Luma) *SysTray {
	return &SysTray{
		lu:      lu,
		closing: make(chan struct{}, 1),
		mu:      &sync.RWMutex{},
	}
}

func (st *SysTray) TrayReady(ctx context.Context) func() {
	return func() {
		systray.SetTemplateIcon(icon.Data, icon.Data)
		systray.SetTooltip("Luma")
		st.StartLocalService(ctx)
	}
}

func (st *SysTray) StartLocalService(ctx context.Context) {
	st.mu.RLock()
	defer st.mu.RUnlock()
	lu := st.lu
	if err := lu.Start(ctx); err != nil {
		log.Errorf("unable to create luma: %v", err)
		os.Exit(1)
	}
}

func (st *SysTray) AddExitMenu() *systray.MenuItem {
	quit := systray.AddMenuItem("Exit", "Exit App")

	go func() {
		for {
			select {
			case <-quit.ClickedCh:
				systray.Quit()
			case <-st.closing:
				return
			}
		}
	}()

	return quit
}

func (st *SysTray) CloseService() {
	st.mu.Lock()
	defer st.mu.Unlock()
	/*if st.browserMenu.Checked() {
		if err := st.ss.SetSysProxyOffHTTP(); err != nil {
			log.Error("[SYSTRAY] close service: set sysproxy off http", "err", err)
		}
	}*/
	if err := st.lu.Stop(); err != nil {
		log.Errorf("close service: close luma: %v", err)
	}
}

func (st *SysTray) Exit() {
	st.closing <- struct{}{}
	st.CloseService()
	log.Info("systray exiting...")
}
