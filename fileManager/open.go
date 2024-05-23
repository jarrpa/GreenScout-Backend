package filemanager

import "os"

func OpenWithPermissions(filepath string) (*os.File, error) {
	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	os.Chmod(filepath, 0777)
	return file, err
}

func MkDirWithPermissions(filepath string) error {
	err := os.Mkdir(filepath, os.ModePerm)
	os.Chmod(filepath, 0777)
	return err
}
