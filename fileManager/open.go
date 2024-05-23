package filemanager

import "os"

func OpenWithPermissions(filepath string) (*os.File, error) {
	return os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
}
