package box

import (
	"os"
	"runtime/debug"
)

var (
	sBasePath    string
	sWorkingPath string
	sTempPath    string
	sUserID      int
	sGroupID     int
	sTVOS        bool
)

func init() {
	debug.SetPanicOnFault(true)
	debug.SetTraceback("all")
}

func Setup(basePath string, workingPath string, tempPath string, isTVOS bool) {
	sBasePath = basePath
	sWorkingPath = workingPath
	sTempPath = tempPath
	sUserID = os.Getuid()
	sGroupID = os.Getgid()
	sTVOS = isTVOS
	os.MkdirAll(sWorkingPath, 0o777)
	os.MkdirAll(sTempPath, 0o777)
}
