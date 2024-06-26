package tun

import "gvisor.dev/gvisor/pkg/tcpip/stack"

// Device is the interface that implemented by network layer devices
type Device interface {
	stack.LinkEndpoint

	// Name returns the current name of the device.
	Name() string

	// Type returns the driver type of the device.
	Type() string
}
