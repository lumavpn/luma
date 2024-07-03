package monitor

import (
	"errors"
	"net/netip"

	"github.com/lumavpn/luma/common/generics/list"
)

var ErrNoRoute = errors.New("no route to internet")

type (
	NetworkUpdateCallback          = func()
	DefaultInterfaceUpdateCallback = func(event int)
)

const (
	EventInterfaceUpdate  = 1
	EventAndroidVPNUpdate = 2
	EventNoRoute          = 4
)

type NetworkUpdateMonitor interface {
	Start() error
	Close() error
	RegisterCallback(callback NetworkUpdateCallback) *list.Element[NetworkUpdateCallback]
	UnregisterCallback(element *list.Element[NetworkUpdateCallback])
}

type DefaultInterfaceMonitor interface {
	Start() error
	Close() error
	DefaultInterfaceName(destination netip.Addr) string
	DefaultInterfaceIndex(destination netip.Addr) int
	DefaultInterface(destination netip.Addr) (string, int)
	OverrideAndroidVPN() bool
	AndroidVPNEnabled() bool
	RegisterCallback(callback DefaultInterfaceUpdateCallback) *list.Element[DefaultInterfaceUpdateCallback]
	UnregisterCallback(element *list.Element[DefaultInterfaceUpdateCallback])
}

type DefaultInterfaceMonitorOptions struct {
	OverrideAndroidVPN    bool
	UnderNetworkExtension bool
}
