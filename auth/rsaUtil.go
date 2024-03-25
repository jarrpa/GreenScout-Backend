package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"log"
	"os"
)

func GetPublicKey() string {
	pubFile, err := os.Open("auth/rsa/login-key.pub.pem")
	if err != nil {
		log.Println(err)
	}
	defer pubFile.Close()

	keyBytes, _ := io.ReadAll(pubFile)

	return string(keyBytes) //Yes this only actually returns the file contents but it'll decode on the frontend
}

// Decrypts the RSA-encoded password
func DecryptPassword(passwordEncrypted []byte) string {
	privFile, _ := os.Open("auth/rsa/login-key.pem")
	defer privFile.Close()

	keyBytes, _ := io.ReadAll(privFile)

	block, _ := pem.Decode(keyBytes)

	key, _ := x509.ParsePKCS1PrivateKey(block.Bytes)

	decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, key, passwordEncrypted)
	if err != nil {
		println("Err: " + err.Error())
	}

	return string(decrypted)
}
