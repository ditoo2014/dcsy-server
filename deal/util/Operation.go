package util

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

/* 设置工作路径 */
func Chdir(workPath string) (err error) {
	dir, err := filepath.Abs(workPath)
	if err != nil {
		return
	}

	err = os.Chdir(dir)
	return
}

func ReadFile(filePth string) ([]byte, error) {
	f, err := os.Open(filePth)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(f)
}
