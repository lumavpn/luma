//go:build !android

package stack

import "os"

func NewPackageManager(callback PackageManagerCallback) (PackageManager, error) {
	return nil, os.ErrInvalid
}
