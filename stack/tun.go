package stack

import (
	"io"
	"net"
	"net/netip"
	"runtime"
	"strconv"
	"strings"

	"github.com/lumavpn/luma/common/errors"
	N "github.com/lumavpn/luma/common/network"
	"github.com/lumavpn/luma/common/ranges"
	"github.com/lumavpn/luma/stack/monitor"
	"github.com/lumavpn/luma/util"
)

type Handler interface {
	N.TCPConnectionHandler
	N.UDPConnectionHandler
	errors.ErrorHandler
}

type Tun interface {
	io.ReadWriter
	N.VectorisedWriter
	Close() error
}

type WinTun interface {
	Tun
	ReadPacket() ([]byte, func(), error)
}

type LinuxTUN interface {
	Tun
	N.FrontHeadroom
	BatchSize() int
	BatchRead(buffers [][]byte, offset int, readN []int) (n int, err error)
	BatchWrite(buffers [][]byte, offset int) error
	TXChecksumOffload() bool
}

type Options struct {
	Name                     string
	Inet4Address             []netip.Prefix
	Inet6Address             []netip.Prefix
	MTU                      uint32
	GSO                      bool
	AutoRoute                bool
	StrictRoute              bool
	Inet4RouteAddress        []netip.Prefix
	Inet6RouteAddress        []netip.Prefix
	Inet4RouteExcludeAddress []netip.Prefix
	Inet6RouteExcludeAddress []netip.Prefix
	IncludeInterface         []string
	ExcludeInterface         []string
	IncludeUID               []ranges.Range[uint32]
	ExcludeUID               []ranges.Range[uint32]
	IncludeAndroidUser       []int
	IncludePackage           []string
	ExcludePackage           []string
	InterfaceMonitor         monitor.DefaultInterfaceMonitor
	TableIndex               int
	FileDescriptor           int

	// No work for TCP, do not use.
	_TXChecksumOffload bool
}

func CalculateInterfaceName(name string) (tunName string) {
	if runtime.GOOS == "darwin" {
		tunName = "utun"
	} else if name != "" {
		tunName = name
	} else {
		tunName = "tun"
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return
	}
	var tunIndex int
	for _, netInterface := range interfaces {
		if strings.HasPrefix(netInterface.Name, tunName) {
			index, parseErr := strconv.ParseInt(netInterface.Name[len(tunName):], 10, 16)
			if parseErr == nil {
				tunIndex = int(index) + 1
			}
		}
	}
	tunName = util.ToString(tunName, tunIndex)
	return
}
