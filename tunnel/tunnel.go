package tunnel

import "github.com/lumavpn/luma/common/atomic"

type tunnel struct {
	status atomic.TypedValue[TunnelStatus]
}

type Tunnel interface {
	// SetStatus sets the current status of the Tunnel
	SetStatus(s TunnelStatus)
	// Status returns the current status of the Tunnel
	Status() TunnelStatus
}

// New returns a new instance of Tunnel
func New() Tunnel {
	return &tunnel{status: atomic.NewTypedValue[TunnelStatus](Disconnected)}
}

// SetStatus sets the current status of the Tunnel
func (t *tunnel) SetStatus(s TunnelStatus) {
	t.status.Store(s)
}

// Status returns the current status of the Tunnel
func (t *tunnel) Status() TunnelStatus {
	return t.status.Load()
}
