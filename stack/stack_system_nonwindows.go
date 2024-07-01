//go:build !windows

package stack

func fixWindowsFirewall() error {
	return nil
}
