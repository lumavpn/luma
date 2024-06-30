package util

import (
	"encoding/binary"
	"net/netip"
)

// UnMasked returns p's last IP address.
// If p is invalid, UnMasked returns netip.Addr{}
func UnMasked(p netip.Prefix) netip.Addr {
	if !p.IsValid() {
		return netip.Addr{}
	}

	buf := p.Addr().As16()

	hi := binary.BigEndian.Uint64(buf[:8])
	lo := binary.BigEndian.Uint64(buf[8:])

	bits := p.Bits()
	if bits <= 32 {
		bits += 96
	}

	hi = hi | ^uint64(0)>>bits
	lo = lo | ^(^uint64(0) << (128 - bits))

	binary.BigEndian.PutUint64(buf[:8], hi)
	binary.BigEndian.PutUint64(buf[8:], lo)

	addr := netip.AddrFrom16(buf)
	if p.Addr().Is4() {
		return addr.Unmap()
	}
	return addr
}
