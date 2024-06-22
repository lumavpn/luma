//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package resolver

import _ "unsafe"

//go:linkname defaultNS net.defaultNS
var defaultNS []string

func init() {
	defaultNS = []string{"8.8.8.8:53"}
}
