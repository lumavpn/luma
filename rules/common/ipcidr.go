package common

import (
	"net/netip"

	M "github.com/lumavpn/luma/metadata"
	R "github.com/lumavpn/luma/rule"
)

type IPCIDROption func(*IPCIDR)

func WithIPCIDRSourceIP(b bool) IPCIDROption {
	return func(i *IPCIDR) {
		i.isSourceIP = b
	}
}

func WithIPCIDRNoResolve(noResolve bool) IPCIDROption {
	return func(i *IPCIDR) {
		i.noResolveIP = noResolve
	}
}

type IPCIDR struct {
	*Base
	ipnet       netip.Prefix
	adapter     string
	isSourceIP  bool
	noResolveIP bool
}

func (i *IPCIDR) RuleType() R.RuleType {
	if i.isSourceIP {
		return R.SrcIPCIDR
	}
	return R.IPCIDR
}

func (i *IPCIDR) Match(metadata *M.Metadata) (bool, string) {
	ip := metadata.DstIP
	if i.isSourceIP {
		ip = metadata.SrcIP
	}
	return ip.IsValid() && i.ipnet.Contains(ip), i.adapter
}

func (i *IPCIDR) Adapter() string {
	return i.adapter
}

func (i *IPCIDR) Payload() string {
	return i.ipnet.String()
}

func (i *IPCIDR) ShouldResolveIP() bool {
	return !i.noResolveIP
}

func NewIPCIDR(s string, adapter string, opts ...IPCIDROption) (*IPCIDR, error) {
	ipnet, err := netip.ParsePrefix(s)
	if err != nil {
		return nil, errPayload
	}

	ipcidr := &IPCIDR{
		Base:    &Base{},
		ipnet:   ipnet,
		adapter: adapter,
	}

	for _, o := range opts {
		o(ipcidr)
	}

	return ipcidr, nil
}

//var _ C.Rule = (*IPCIDR)(nil)
