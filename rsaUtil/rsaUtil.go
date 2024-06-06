package rsaUtil

// Utility to handle RSA encryption/decription

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

// Returns the contents of the public key used for password encryption
func GetPublicKey() string {
	pubFile, openErr := os.Open(filepath.Join("rsaUtil", "login-key.pub.pem"))
	if openErr != nil {
		greenlogger.LogErrorf(openErr, "Problem opening %v", filepath.Join("rsaUtil", "login-key.pub.pem"))
		return ""
	}
	defer pubFile.Close()

	keyBytes, readErr := io.ReadAll(pubFile)
	if readErr != nil {
		greenlogger.LogErrorf(readErr, "Problem reading %v", filepath.Join("rsaUtil", "login-key.pub.pem"))
		return ""
	}

	return string(keyBytes) //Yes this only actually returns the file contents but it'll decode on the frontend
}

// Decrypts the RSA-encoded password
func DecryptPassword(passwordEncrypted []byte) string {
	if len(passwordEncrypted) == 0 {
		return ""
	}

	privFile, openErr := os.Open(filepath.Join("rsaUtil", "login-key.pem"))
	if openErr != nil {
		greenlogger.LogErrorf(openErr, "Problem opening %v", filepath.Join("rsaUtil", "login-key.pem"))
		return ""
	}

	defer privFile.Close()

	keyBytes, readErr := io.ReadAll(privFile)
	if readErr != nil {
		greenlogger.LogErrorf(readErr, "Problem reading %v", filepath.Join("rsaUtil", "login-key.pem"))
		return ""
	}

	block, _ := pem.Decode(keyBytes)

	key, parseErr := x509.ParsePKCS1PrivateKey(block.Bytes)
	if parseErr != nil {
		greenlogger.LogErrorf(parseErr, "Problem parsing %v", block.Bytes)
		return ""
	}

	decrypted, decryptErr := rsa.DecryptPKCS1v15(rand.Reader, key, passwordEncrypted)
	if decryptErr != nil {
		greenlogger.LogErrorf(decryptErr, "Problem decrypting %v", passwordEncrypted)
		return ""
	}

	return string(decrypted)
}

// Encodes a message with the public key
func EncodeWithPublicKey(message string) []byte {
	pubFile, _ := os.Open(filepath.Join("rsaUtil", "login-key.pub.pem"))
	defer pubFile.Close()

	keyBytes, readErr := io.ReadAll(pubFile)
	if readErr != nil {
		greenlogger.LogErrorf(readErr, "Problem reading %v", filepath.Join("rsaUtil", "login-key.pub.pem"))
		return []byte("")
	}

	block, _ := pem.Decode(keyBytes)

	key, parseErr := x509.ParsePKCS1PublicKey(block.Bytes)

	if parseErr != nil {
		greenlogger.LogErrorf(parseErr, "Problem parsing %v", block.Bytes)
		return []byte("")
	}

	result, encryptErr := rsa.EncryptPKCS1v15(rand.Reader, key, []byte(message))
	if encryptErr != nil {
		greenlogger.LogErrorf(encryptErr, "Problem encrypting %v", message)
		return []byte("")
	}

	return result
}
