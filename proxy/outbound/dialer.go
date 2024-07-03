package outbound

/*import (
	"context"
	"time"

	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/metadata"
	C "github.com/lumavpn/luma/proxy"
)

const (
	tcpConnectTimeout = 5 * time.Second
)

var _defaultDialer Dialer = &Base{}

// Dialer provides information needed to dial a proxy and effectively load balancer
// between dialers
type Dialer interface {
	DialContext(context.Context, *metadata.Metadata, ...dialer.Option) (C.Conn, error)
	ListenPacketContext(context.Context, *metadata.Metadata, ...dialer.Option) (C.PacketConn, error)
}

// SetDialer sets default Dialer.
func SetDialer(d Dialer) {
	_defaultDialer = d
}

// Dial uses default Dialer to dial TCP.
func Dial(metadata *metadata.Metadata) (C.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), tcpConnectTimeout)
	defer cancel()
	return _defaultDialer.DialContext(ctx, metadata)
}

// DialContext uses default Dialer to dial TCP with context.
func DialContext(ctx context.Context, metadata *metadata.Metadata) (C.Conn, error) {
	return _defaultDialer.DialContext(ctx, metadata)
}

// DialUDP uses default Dialer to dial UDP.
func ListenPacketContext(ctx context.Context, metadata *metadata.Metadata, opts ...dialer.Option) (C.PacketConn, error) {
	return _defaultDialer.ListenPacketContext(ctx, metadata, opts...)
}*/
