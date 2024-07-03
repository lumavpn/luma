package stack

import "github.com/lumavpn/luma/common/errors"

type PackageManager interface {
	Start() error
	Close() error
	IDByPackage(packageName string) (uint32, bool)
	IDBySharedPackage(sharedPackage string) (uint32, bool)
	PackageByID(id uint32) (string, bool)
	SharedPackageByID(id uint32) (string, bool)
}

type PackageManagerCallback interface {
	OnPackagesUpdated(packages int, sharedUsers int)
	errors.ErrorHandler
}
