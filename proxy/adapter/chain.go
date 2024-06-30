package adapter

import "fmt"

type Chain []string

func (c Chain) String() string {
	switch len(c) {
	case 0:
		return ""
	case 1:
		return c[0]
	default:
		return fmt.Sprintf("%s[%s]", c[len(c)-1], c[0])
	}
}

func (c Chain) Last() string {
	switch len(c) {
	case 0:
		return ""
	default:
		return c[0]
	}
}
