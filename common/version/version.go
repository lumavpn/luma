package version

import (
	"fmt"
	"strings"
)

const AppName = "luma"

var (
	Version   string
	GitCommit string
)

func versionize(s string) string {
	return strings.TrimPrefix(s, "v")
}

func String() string {
	return fmt.Sprintf("%s-%s (debug)", AppName, versionize(Version))
}
