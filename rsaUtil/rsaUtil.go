package rsaUtil

// Utility to handle RSA encryption/decription

import (
	"GreenScoutBackend/constants"
	greenlogger "GreenScoutBackend/greenLogger"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
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

func GenLocalhostCert() (string, string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", "", err
	}
	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	// PEM encoding of private key
	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: keyBytes,
		},
	)
	fmt.Println(string(keyPEM))

	//Create certificate templet
	template := x509.Certificate{
		SerialNumber:          big.NewInt(0),
		Subject:               pkix.Name{CommonName: "localhost"},
		DNSNames:              []string{"localhost"},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement | x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	//Create certificate using templet
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return "", "", err

	}
	//pem encoding of certificate
	certPem := string(pem.EncodeToMemory(
		&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: derBytes,
		},
	))
	fmt.Println(certPem)

	return string(keyPEM), certPem, nil
}
