package rsaUtil

import (
	greenlogger "GreenScoutBackend/greenLogger"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"os"
	"path/filepath"
)

func GetPublicKey() string {
	pubFile, openErr := os.Open(filepath.Join("rsaUtil", "login-key.pub.pem"))
	if openErr != nil {
		greenlogger.LogErrorf(openErr, "Problem opening %v", filepath.Join("rsaUtil", "login-key.pub.pem"))
		// SLACK INTEGRATE
		return ""
	}
	defer pubFile.Close()

	keyBytes, readErr := io.ReadAll(pubFile)
	if readErr != nil {
		greenlogger.LogErrorf(readErr, "Problem reading %v", filepath.Join("rsaUtil", "login-key.pub.pem"))
		// SLACK INTEGRATE
		return ""
	}

	return string(keyBytes) //Yes this only actually returns the file contents but it'll decode on the frontend
}

// Decrypts the RSA-encoded password
func DecryptPassword(passwordEncrypted []byte) string {
	privFile, openErr := os.Open(filepath.Join("rsaUtil", "login-key.pem"))
	if openErr != nil {
		greenlogger.LogErrorf(openErr, "Problem opening %v", filepath.Join("rsaUtil", "login-key.pem"))
		// SLACK INTEGRATE
		return ""
	}

	defer privFile.Close()

	keyBytes, readErr := io.ReadAll(privFile)
	if readErr != nil {
		greenlogger.LogErrorf(readErr, "Problem reading %v", filepath.Join("rsaUtil", "login-key.pem"))
		// SLACK INTEGRATE
		return ""
	}

	block, _ := pem.Decode(keyBytes)

	key, parseErr := x509.ParsePKCS1PrivateKey(block.Bytes)
	if parseErr != nil {
		greenlogger.LogErrorf(parseErr, "Problem parsing %v", block.Bytes)
		// SLACK INTEGRATE
		return ""
	}

	decrypted, decryptErr := rsa.DecryptPKCS1v15(rand.Reader, key, passwordEncrypted)
	if decryptErr != nil {
		greenlogger.LogErrorf(decryptErr, "Problem decrypting %v", passwordEncrypted)
		// SLACK INTEGRATE
		return ""
	}

	return string(decrypted)
}

func EncodeWithPublicKey(message string) []byte {
	pubFile, _ := os.Open(filepath.Join("rsaUtil", "login-key.pub.pem"))
	defer pubFile.Close()

	keyBytes, readErr := io.ReadAll(pubFile)
	if readErr != nil {
		greenlogger.LogErrorf(readErr, "Problem reading %v", filepath.Join("rsaUtil", "login-key.pub.pem"))
		// SLACK INTEGRATE
		return []byte("")
	}

	block, _ := pem.Decode(keyBytes)

	key, parseErr := x509.ParsePKCS1PublicKey(block.Bytes)

	if parseErr != nil {
		greenlogger.LogErrorf(parseErr, "Problem parsing %v", block.Bytes)
		// SLACK INTEGRATE
		return []byte("")
	}

	result, encryptErr := rsa.EncryptPKCS1v15(rand.Reader, key, []byte(message))
	if encryptErr != nil {
		greenlogger.LogErrorf(encryptErr, "Problem encrypting %v", message)
		// SLACK INTEGRATE
		return []byte("")
	}

	return result
}