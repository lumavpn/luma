//go:build !with_low_memory

package pool

const (
	RelayBufferSize = 20 * 1024

	UDPBufferSize = 16 * 1024
)
