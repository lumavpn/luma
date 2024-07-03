package common

import "time"

const (
	DefaultTCPTimeout = 5 * time.Second
	DefaultUDPTimeout = DefaultTCPTimeout
	DefaultDropTime   = 12 * DefaultTCPTimeout
	DefaultTLSTimeout = DefaultTCPTimeout
	DefaultTestURL    = "https://www.gstatic.com/generate_204"

	LumaName = "Luma"
)
