package dns

type DNSPrefer int

const (
	DualStack DNSPrefer = iota
	IPv4Only
	IPv6Only
	IPv4Prefer
	IPv6Prefer
)

var dnsPreferMap = map[string]DNSPrefer{
	DualStack.String():  DualStack,
	IPv4Only.String():   IPv4Only,
	IPv6Only.String():   IPv6Only,
	IPv4Prefer.String(): IPv4Prefer,
	IPv6Prefer.String(): IPv6Prefer,
}

func (d DNSPrefer) String() string {
	switch d {
	case DualStack:
		return "dual"
	case IPv4Only:
		return "ipv4"
	case IPv6Only:
		return "ipv6"
	case IPv4Prefer:
		return "ipv4-prefer"
	case IPv6Prefer:
		return "ipv6-prefer"
	default:
		return "dual"
	}
}

func NewDNSPrefer(prefer string) DNSPrefer {
	if p, ok := dnsPreferMap[prefer]; ok {
		return p
	} else {
		return DualStack
	}
}
