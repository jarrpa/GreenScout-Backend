package pfp

import (
	filemanager "GreenScoutBackend/fileManager"
	greenlogger "GreenScoutBackend/greenLogger"
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
)

func CheckForPfp(name string) bool {
	file, err := os.Open(filepath.Join("pfp", "pictures", name))
	file.Close()

	return err == nil
}

func WritePfp(imgBytes []byte, name string) bool {
	file, openErr := filemanager.OpenWithPermissions(filepath.Join("pfp", "pictures", name))

	if openErr != nil {
		greenlogger.LogErrorf(openErr, "Problem opening %v", filepath.Join(filepath.Join("pfp", "pictures", name)))
	}

	defer file.Close()

	image, format, imagErr := image.Decode(bytes.NewReader(imgBytes))

	if imagErr != nil {
		greenlogger.LogError(imagErr, "Problem decoding image")
	}

	var encodeErr error
	if format == "png" {
		encodeErr = png.Encode(file, image)
	} else if format == "jpeg" {
		encodeErr = jpeg.Encode(file, image, &jpeg.Options{Quality: 100})
	} else {
		return false
	}

	if encodeErr != nil {
		greenlogger.LogErrorf(encodeErr, "Problem encoding %v", filepath.Join(filepath.Join("pfp", "pictures", name)))
	}

	return encodeErr == nil
}
