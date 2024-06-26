package version

import (
	"fmt"
	"runtime"
	"strings"
)

const AppName = "luma"

var (
	_debug = false

	// Version can be set at link time by executing
	// the command: `git describe --abbrev=0 --tags HEAD`
	Version string

	// GitCommit can be set at link time by executing
	// the command: `git rev-parse --short HEAD`
	GitCommit string
)

func versionize(s string) string {
	return strings.TrimPrefix(s, "v")
}

func Debug() bool {
	return _debug
}

func String() string {
	if !Debug() {
		return fmt.Sprintf("%s-%s", AppName, versionize(Version))
	}
	return fmt.Sprintf("%s-%s (debug)", AppName, versionize(Version))
}

func BuildString() string {
	return fmt.Sprintf("%s/%s, %s, %s", runtime.GOOS, runtime.GOARCH, runtime.Version(), GitCommit)
}
