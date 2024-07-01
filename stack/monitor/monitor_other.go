//go:build !(linux || windows || darwin)

package monitor

import (
	"os"
)

func NewNetworkUpdateMonitor() (NetworkUpdateMonitor, error) {
	return nil, os.ErrInvalid
}

func NewDefaultInterfaceMonitor(networkMonitor NetworkUpdateMonitor, options DefaultInterfaceMonitorOptions) (DefaultInterfaceMonitor, error) {
	return nil, os.ErrInvalid
}
