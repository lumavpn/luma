package tun

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"runtime"
	"strconv"
	"strings"

	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/common/errors"
	"github.com/lumavpn/luma/common/ranges"
	"github.com/lumavpn/luma/component/iface"
	"github.com/lumavpn/luma/component/resolver"
	"github.com/lumavpn/luma/config"
	"github.com/lumavpn/luma/dialer"
	"github.com/lumavpn/luma/listener/mux"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy/inbound"
	"github.com/lumavpn/luma/proxy/proto"
	"github.com/lumavpn/luma/stack"
	"github.com/lumavpn/luma/util"
)

var InterfaceName = "Meta"

type Listener struct {
	closed  bool
	options *config.Tun
	handler *ListenerHandler
	tunName string
	addrStr string

	tunIf    stack.Tun
	tunStack stack.Stack

	//packageManager stack.PackageManager

	dnsServerIp []string
}

func CalculateInterfaceName(name string) (tunName string) {
	if runtime.GOOS == "darwin" {
		tunName = "utun"
	} else if name != "" {
		tunName = name
		return
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

func checkTunName(tunName string) (ok bool) {
	defer func() {
		if !ok {
			log.Warnf("[TUN] Unsupported tunName(%s) in %s, force regenerate by ourselves.", tunName, runtime.GOOS)
		}
	}()
	log.Debugf("tun name is %s", tunName)
	if runtime.GOOS == "darwin" {
		if len(tunName) <= 4 {
			return false
		}
		if tunName[:4] != "utun" {
			return false
		}
		if _, parseErr := strconv.ParseInt(tunName[4:], 10, 16); parseErr != nil {
			return false
		}
	}
	return true
}

func New(options *config.Tun, tunnel adapter.TransportHandler, additions ...inbound.Option) (l *Listener, err error) {
	if len(additions) == 0 {
		additions = []inbound.Option{
			inbound.WithInName("DEFAULT-TUN"),
			inbound.WithSpecialRules(""),
		}
	}
	if options.GSOMaxSize == 0 {
		options.GSOMaxSize = 65536
	}
	tunName := options.Device
	if tunName == "" || !checkTunName(tunName) {
		tunName = CalculateInterfaceName(InterfaceName)
		log.Debugf("Setting tun device name to %s", tunName)
		options.Device = tunName
	}
	tunMTU := options.MTU
	if tunMTU == 0 {
		tunMTU = 9000
	}
	var udpTimeout int64
	if options.UDPTimeout != 0 {
		udpTimeout = options.UDPTimeout
	} else {
		udpTimeout = int64(mux.UDPTimeout.Seconds())
	}
	tableIndex := options.TableIndex
	if tableIndex == 0 {
		tableIndex = 2022
	}
	includeUID := uidToRange(options.IncludeUID)
	if len(options.IncludeUIDRange) > 0 {
		var err error
		includeUID, err = parseRange(includeUID, options.IncludeUIDRange)
		if err != nil {
			return nil, errors.Cause(err, "parse include_uid_range")
		}
	}
	excludeUID := uidToRange(options.ExcludeUID)
	if len(options.ExcludeUIDRange) > 0 {
		var err error
		excludeUID, err = parseRange(excludeUID, options.ExcludeUIDRange)
		if err != nil {
			return nil, errors.Cause(err, "parse exclude_uid_range")
		}
	}

	var dnsAdds []netip.AddrPort

	for _, d := range options.DNSHijack {
		if _, after, ok := strings.Cut(d, "://"); ok {
			d = after
		}
		d = strings.Replace(d, "any", "0.0.0.0", 1)
		addrPort, err := netip.ParseAddrPort(d)
		if err != nil {
			return nil, fmt.Errorf("parse dns-hijack url error: %w", err)
		}

		dnsAdds = append(dnsAdds, addrPort)
	}

	var dnsServerIp []string
	for _, a := range options.Inet4Address {
		addrPort := netip.AddrPortFrom(a.Addr().Next(), 53)
		dnsServerIp = append(dnsServerIp, a.Addr().Next().String())
		dnsAdds = append(dnsAdds, addrPort)
	}
	for _, a := range options.Inet6Address {
		addrPort := netip.AddrPortFrom(a.Addr().Next(), 53)
		dnsServerIp = append(dnsServerIp, a.Addr().Next().String())
		dnsAdds = append(dnsAdds, addrPort)
	}

	h, err := mux.NewListener(mux.Options{
		Tunnel:    tunnel,
		Type:      proto.Proto_Tun,
		Additions: additions,
	})
	if err != nil {
		return nil, err
	}

	handler := &ListenerHandler{
		Listener: h,
		DnsAdds:  dnsAdds,
	}
	l = &Listener{
		closed:  false,
		options: options,
		handler: handler,
		tunName: tunName,
	}
	defer func() {
		if err != nil {
			l.Close()
			l = nil
		}
	}()

	tunOptions := stack.Options{
		Name:                     tunName,
		MTU:                      tunMTU,
		GSO:                      options.GSO,
		Inet4Address:             options.Inet4Address,
		Inet6Address:             options.Inet6Address,
		AutoRoute:                options.AutoRoute,
		StrictRoute:              options.StrictRoute,
		Inet4RouteAddress:        options.Inet4RouteAddress,
		Inet6RouteAddress:        options.Inet6RouteAddress,
		Inet4RouteExcludeAddress: options.Inet4RouteExcludeAddress,
		Inet6RouteExcludeAddress: options.Inet6RouteExcludeAddress,
		IncludeInterface:         options.IncludeInterface,
		ExcludeInterface:         options.ExcludeInterface,
		IncludeUID:               includeUID,
		ExcludeUID:               excludeUID,
		IncludeAndroidUser:       options.IncludeAndroidUser,
		IncludePackage:           options.IncludePackage,
		ExcludePackage:           options.ExcludePackage,
		FileDescriptor:           options.FileDescriptor,
		TableIndex:               tableIndex,
	}

	if options.BuildAndroidRules {
		err = l.buildAndroidRules(&tunOptions)
		if err != nil {
			err = errors.Cause(err, "build android rules")
			return
		}
	}
	tunIf, err := tunNew(tunOptions)
	if err != nil {
		err = errors.Cause(err, "configure tun interface")
		return
	}

	l.dnsServerIp = dnsServerIp
	// after tun.New sing-tun has set DNS to TUN interface
	resolver.AddSystemDnsBlacklist(dnsServerIp...)

	stackOptions := &stack.Config{
		Context:                context.Background(),
		Stack:                  options.Stack,
		Tun:                    tunIf,
		TunOptions:             tunOptions,
		EndpointIndependentNat: options.EndpointIndependentNat,
		UDPTimeout:             udpTimeout,
		Handler:                handler,
	}

	if options.FileDescriptor > 0 {
		if tunName, err := getTunnelName(int32(options.FileDescriptor)); err != nil {
			stackOptions.TunOptions.Name = tunName
			stackOptions.ForwarderBindInterface = true
		}
	}
	l.tunIf = tunIf

	log.Debug("Creating new stack..")
	tunStack, err := stack.NewStack(stackOptions)
	if err != nil {
		return
	}
	log.Debug("Starting stack..")
	err = tunStack.Start(context.Background())
	if err != nil {
		return
	}
	l.tunStack = tunStack

	//l.openAndroidHotspot(tunOptions)

	l.addrStr = fmt.Sprintf("%s(%s,%s), mtu: %d, auto route: %v, ip stack: %s",
		tunName, tunOptions.Inet4Address, tunOptions.Inet6Address, tunMTU, options.AutoRoute, options.Stack)
	return
}

func (l *Listener) FlushDefaultInterface() {
	if l.options.AutoDetectInterface {
		targetInterface := dialer.DefaultInterface.Load()
		for _, destination := range []netip.Addr{netip.IPv4Unspecified(), netip.IPv6Unspecified(), netip.MustParseAddr("1.1.1.1")} {
			//autoDetectInterfaceName := l.defaultInterfaceMonitor.DefaultInterfaceName(destination)
			autoDetectInterfaceName := ""
			if autoDetectInterfaceName == l.tunName {
				log.Warnf("[TUN] Auto detect interface by %s get same name with tun", destination.String())
			} else if autoDetectInterfaceName == "" || autoDetectInterfaceName == "<nil>" {
				log.Warnf("[TUN] Auto detect interface by %s get empty name.", destination.String())
			} else {
				targetInterface = autoDetectInterfaceName
				if old := dialer.DefaultInterface.Load(); old != targetInterface {
					log.Warnf("[TUN] default interface changed by monitor, %s => %s", old, targetInterface)

					dialer.DefaultInterface.Store(targetInterface)

					iface.FlushCache()
				}
				return
			}
		}
	}
}

func uidToRange(uidList []uint32) []ranges.Range[uint32] {
	return util.Map(uidList, func(uid uint32) ranges.Range[uint32] {
		return ranges.NewSingle(uid)
	})
}

func parseRange(uidRanges []ranges.Range[uint32], rangeList []string) ([]ranges.Range[uint32], error) {
	for _, uidRange := range rangeList {
		if !strings.Contains(uidRange, ":") {
			return nil, errors.New("missing ':' in range: ", uidRange)
		}
		subIndex := strings.Index(uidRange, ":")
		if subIndex == 0 {
			return nil, errors.New("missing range start: ", uidRange)
		} else if subIndex == len(uidRange)-1 {
			return nil, errors.New("missing range end: ", uidRange)
		}
		var start, end uint64
		var err error
		start, err = strconv.ParseUint(uidRange[:subIndex], 10, 32)
		if err != nil {
			return nil, errors.Cause(err, "parse range start")
		}
		end, err = strconv.ParseUint(uidRange[subIndex+1:], 10, 32)
		if err != nil {
			return nil, errors.Cause(err, "parse range end")
		}
		uidRanges = append(uidRanges, ranges.New(uint32(start), uint32(end)))
	}
	return uidRanges, nil
}

func (l *Listener) Close() error {
	l.closed = true
	return util.Close(
		l.tunStack,
		l.tunIf,
		//l.defaultInterfaceMonitor,
		//l.networkUpdateMonitor,
	)
}

func (l *Listener) Config() *config.Tun {
	return l.options
}

func (l *Listener) Address() string {
	return l.addrStr
}
