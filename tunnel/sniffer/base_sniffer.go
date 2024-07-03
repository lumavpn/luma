package sniffer

import (
	"errors"

	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/util"
)

type SnifferConfig struct {
	OverrideDest bool
	Ports        util.IntRanges[uint16]
}

type BaseSniffer struct {
	ports              util.IntRanges[uint16]
	supportNetworkType M.Network
}

// Protocol implements Sniffer
func (*BaseSniffer) Protocol() string {
	return "unknown"
}

// SniffData implements Sniffer
func (*BaseSniffer) SniffData(bytes []byte) (string, error) {
	return "", errors.New("TODO")
}

// SupportNetwork implements Sniffer
func (bs *BaseSniffer) SupportNetwork() M.Network {
	return bs.supportNetworkType
}

// SupportPort implements Sniffer
func (bs *BaseSniffer) SupportPort(port uint16) bool {
	return bs.ports.Check(port)
}

func NewBaseSniffer(ports util.IntRanges[uint16], networkType M.Network) *BaseSniffer {
	return &BaseSniffer{
		ports:              ports,
		supportNetworkType: networkType,
	}
}

var _ Sniffer = (*BaseSniffer)(nil)
