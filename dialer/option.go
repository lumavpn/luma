package dialer

type option struct {
	interfaceName string
	routingMark   int32
}

type Option func(opt *option)

// WithInterface sets the name of the interface to dial with
func WithInterface(name string) Option {
	return func(opt *option) {
		opt.interfaceName = name
	}
}

// WithRoutingMark updates the routing mark option
func WithRoutingMark(mark int32) Option {
	return func(opt *option) {
		opt.routingMark = mark
	}
}
