package monitor

import (
	"sync"

	E "github.com/lumavpn/luma/common/errors"
	"github.com/lumavpn/luma/common/generics/list"
	"github.com/lumavpn/luma/stack/internal/winipcfg"

	"golang.org/x/sys/windows"
)

type networkUpdateMonitor struct {
	routeListener     *winipcfg.RouteChangeCallback
	interfaceListener *winipcfg.InterfaceChangeCallback
	errorHandler      E.ErrorHandler

	access    sync.Mutex
	callbacks list.List[NetworkUpdateCallback]
}

func NewNetworkUpdateMonitor() (NetworkUpdateMonitor, error) {
	return &networkUpdateMonitor{}, nil
}

func (m *networkUpdateMonitor) Start() error {
	routeListener, err := winipcfg.RegisterRouteChangeCallback(func(notificationType winipcfg.MibNotificationType, route *winipcfg.MibIPforwardRow2) {
		m.emit()
	})
	if err != nil {
		return err
	}
	m.routeListener = routeListener
	interfaceListener, err := winipcfg.RegisterInterfaceChangeCallback(func(notificationType winipcfg.MibNotificationType, iface *winipcfg.MibIPInterfaceRow) {
		m.emit()
	})
	if err != nil {
		routeListener.Unregister()
		return err
	}
	m.interfaceListener = interfaceListener
	return nil
}

func (m *networkUpdateMonitor) Close() error {
	if m.routeListener != nil {
		m.routeListener.Unregister()
		m.routeListener = nil
	}
	if m.interfaceListener != nil {
		m.interfaceListener.Unregister()
		m.interfaceListener = nil
	}
	return nil
}

func (m *defaultInterfaceMonitor) checkUpdate() error {
	rows, err := winipcfg.GetIPForwardTable2(windows.AF_INET)
	if err != nil {
		return err
	}

	lowestMetric := ^uint32(0)
	alias := ""
	var index int

	for _, row := range rows {
		if row.DestinationPrefix.PrefixLength != 0 {
			continue
		}

		ifrow, err := row.InterfaceLUID.Interface()
		if err != nil || ifrow.OperStatus != winipcfg.IfOperStatusUp {
			continue
		}

		iface, err := row.InterfaceLUID.IPInterface(windows.AF_INET)
		if err != nil {
			continue
		}

		if ifrow.Type == winipcfg.IfTypePropVirtual || ifrow.Type == winipcfg.IfTypeSoftwareLoopback {
			continue
		}

		metric := row.Metric + iface.Metric
		if metric < lowestMetric {
			lowestMetric = metric
			alias = ifrow.Alias()
			index = int(ifrow.InterfaceIndex)
		}
	}

	if alias == "" {
		return ErrNoRoute
	}

	oldInterface := m.defaultInterfaceName
	oldIndex := m.defaultInterfaceIndex

	m.defaultInterfaceName = alias
	m.defaultInterfaceIndex = index

	if oldInterface == m.defaultInterfaceName && oldIndex == m.defaultInterfaceIndex {
		return nil
	}

	m.emit(EventInterfaceUpdate)
	return nil
}
