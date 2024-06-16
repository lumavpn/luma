package dialer

type option struct {
	// interfaceName is the name of the interface to bind
	interfaceName string
	// interfaceIndex is the index of the interface to bind
	interfaceIndex int
	// routingMark is the mark for each packet sent through this socket
	routingMark int
}

type Option func(opt *option)

// WithInterface sets the name of the interface to dial with
func WithInterface(name string) Option {
	return func(opt *option) {
		opt.interfaceName = name
	}
}

// WithInterfaceIndex sets the index of the interface to dial with
func WithInterfaceIndex(idx int) Option {
	return func(opt *option) {
		opt.interfaceIndex = idx
	}
}

// WithRoutingMark updates the mark for each packet sent through this socket
func WithRoutingMark(mark int) Option {
	return func(opt *option) {
		opt.routingMark = mark
	}
}
