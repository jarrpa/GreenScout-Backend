package userDB

import (
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func GetCertificate(username string, role string) string {
	var certificate string
	result := userDB.QueryRow("select certificate from users where username = ?", username)
	result.Scan(&certificate)

	_, authenticated := VerifyCertificate(certificate)
	if certificate == "" || !authenticated {
		// Update user db
		newCertRaw := uuid.New() //Using uuid.new because it's a good source of randomness
		newCert, _ := bcrypt.GenerateFromPassword([]byte(newCertRaw.String()), 6)

		_, err := userDB.Exec("update users set certificate = ? where username = ?", string(newCert), username)

		if err != nil {
			println(err.Error())
		}

		certificate = string(newCert)

		// Update certificate db
		authDB.Exec("insert into certs values(?,?,?)", string(newCert), role, username)
	}

	return certificate
}

func VerifyCertificate(certificate string) (string, bool) {
	var certificateRole string
	result := authDB.QueryRow("select role from certs where certificate = ?", certificate)
	newErr := result.Scan(&certificateRole)

	if newErr != nil {
		return "none", false
	}

	return certificateRole, true
}
