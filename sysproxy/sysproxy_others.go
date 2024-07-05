//go:build !windows && !darwin

package sysproxy

func EnableSystemProxy(opts *Opts) error {
	return nil
}

func DisableSystemProxy() {
}

func WebProxySwitch(status bool, args ...string) error {
	return nil
}

func SecureWebProxySwitch(status bool, args ...string) error {
	return nil
}

func SocksProxySwitch(status bool, args ...string) error {
	return nil
}
