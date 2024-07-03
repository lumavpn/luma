package util

import (
	"errors"
	"os"
	"path/filepath"
)

func CurrentDir() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(path)
}

func FileExists(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err == nil {
		if fi.Mode()&os.ModeType == 0 {
			return true, nil
		}
		return false, errors.New(path + " exists but is not regular file")
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
