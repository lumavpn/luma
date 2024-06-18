package stack

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

type NativeTun struct {
	tunFile      *os.File
	mtu          uint32
	inet4Address [4]byte
	inet6Address [16]byte
}

func New(options Options) (Tun, error) {
	var tunFd int
	if options.FileDescriptor == 0 {
		ifIndex := -1
		_, err := fmt.Sscanf(options.Name, "utun%d", &ifIndex)
		if err != nil {
			return nil, fmt.Errorf("bad tun name: %v", options.Name)
		}

		tunFd, err = unix.Socket(unix.AF_SYSTEM, unix.SOCK_DGRAM, 2)
		if err != nil {
			return nil, err
		}

		err = configure(tunFd, ifIndex, options.Name, options)
		if err != nil {
			unix.Close(tunFd)
			return nil, err
		}
	} else {
		tunFd = options.FileDescriptor
	}

	nativeTun := &NativeTun{
		tunFile: os.NewFile(uintptr(tunFd), "utun"),
		mtu:     options.MTU,
	}

	return nativeTun, nil
}

func (t *NativeTun) Read(p []byte) (n int, err error) {
	return t.tunFile.Read(p)
}

func (t *NativeTun) Write(p []byte) (n int, err error) {
	return t.tunFile.Write(p)
}

var (
	packetHeader4 = [4]byte{0x00, 0x00, 0x00, unix.AF_INET}
	packetHeader6 = [4]byte{0x00, 0x00, 0x00, unix.AF_INET6}
)

func (t *NativeTun) Close() error {
	return t.tunFile.Close()
}

func configure(tunFd int, ifIndex int, name string, options Options) error {
	return nil
}
