package dns

type NameServer struct {
	Net       string
	Addr      string
	Interface string
}

func (ns NameServer) Equal(ns2 NameServer) bool {
	return ns.Net == ns2.Net &&
		ns.Addr == ns2.Addr &&
		ns.Interface == ns2.Interface
}
