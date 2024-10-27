package rsaUtil

// Utility to handle RSA encryption/decription

import (
	"GreenScoutBackend/constants"
	greenlogger "GreenScoutBackend/greenLogger"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"os"
)

// Returns the contents of the public key used for password encryption
func GetPublicKey() string {
	pubFile, openErr := os.Open(constants.RSAPubKeyPath)
	if openErr != nil {
		greenlogger.LogErrorf(openErr, "Problem opening %v", constants.RSAPubKeyPath)
		return ""
	}
	defer pubFile.Close()

	keyBytes, readErr := io.ReadAll(pubFile)
	if readErr != nil {
		greenlogger.LogErrorf(readErr, "Problem reading %v", constants.RSAPubKeyPath)
		return ""
	}

	return string(keyBytes) //Yes this only actually returns the file contents but it'll decode on the frontend
}

// Decrypts the RSA-encoded password
func DecryptPassword(passwordEncrypted []byte) string {
	if len(passwordEncrypted) == 0 {
		return ""
	}

	privFile, openErr := os.Open(constants.RSAPrivateKeyPath)
	if openErr != nil {
		greenlogger.LogErrorf(openErr, "Problem opening %v", constants.RSAPrivateKeyPath)
		return ""
	}

	defer privFile.Close()

	keyBytes, readErr := io.ReadAll(privFile)
	if readErr != nil {
		greenlogger.LogErrorf(readErr, "Problem reading %v", constants.RSAPrivateKeyPath)
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
	pubFile, _ := os.Open(constants.RSAPubKeyPath)
	defer pubFile.Close()

	keyBytes, readErr := io.ReadAll(pubFile)
	if readErr != nil {
		greenlogger.LogErrorf(readErr, "Problem reading %v", constants.RSAPubKeyPath)
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
