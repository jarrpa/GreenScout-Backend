package auth

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type LoginAttempt struct {
	Username          string
	EncryptedPassword string
}

func Authenticate(passwordEncoded []byte) (string, bool) {
	passwordPlain := DecryptPassword(passwordEncoded)
	database, err := sql.Open("sqlite3", "./auth.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	checkAgainst := make(map[string]string)

	rows, _ := database.Query("select role, password from roles")

	for rows.Next() {
		var role string
		var hashedWord string
		rows.Scan(&role, &hashedWord)
		checkAgainst[role] = hashedWord
	}

	for role, toCheck := range checkAgainst {
		if comparePasswordBCrypt(passwordPlain, toCheck) {
			return role, true
		}
	}

	return "Not accepted nuh uh", false
}

func GetUUID(username string) string {
	database, err := sql.Open("sqlite3", "./auth.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	if !userExists(database, username) {
		database.Exec(fmt.Sprintf("insert into users values('%s',null, null)", username))
	}

	var userId string
	result := database.QueryRow(fmt.Sprintf("select uuid from users where username  = '%s'", username))
	result.Scan(&userId)

	if userId == "" {
		newId := uuid.New()
		_, err := database.Exec(fmt.Sprintf("update users set uuid = '%v' where username = '%s'", newId, username))

		if err != nil {
			println(err.Error())
		}

		userId = newId.String()
	}
	return userId
}

func userExists(database *sql.DB, username string) bool {
	result := database.QueryRow(fmt.Sprintf("select count(1) from users where username = '%s'", username))

	var resultstore int
	result.Scan(&resultstore)

	return resultstore == 1
}

func comparePasswordBCrypt(plainPassword string, encodedPassword string) bool {
	err := bcrypt.CompareHashAndPassword(
		[]byte(encodedPassword), []byte(plainPassword))

	return err == nil
}

func GetCertificate(username string, role string) string {
	database, err := sql.Open("sqlite3", "./auth.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	var certificate string
	result := database.QueryRow(fmt.Sprintf("select certificate from users where username  = '%s'", username))
	result.Scan(&certificate)

	if certificate == "" {
		// Update user db
		newCertRaw := uuid.New() //Using uuid.new because it's a good source of randomness
		newCert, _ := bcrypt.GenerateFromPassword([]byte(newCertRaw.String()), 6)

		_, err := database.Exec(fmt.Sprintf("update users set certificate = '%v' where username = '%s'", string(newCert), username))

		if err != nil {
			println(err.Error())
		}

		certificate = string(newCert)

		// Update certificate db
		database.Exec(fmt.Sprintf("insert into certificates values('%s','%s')", string(newCert), role))
	}

	return certificate
}

func VerifyCertificate(certificate string) (string, bool) {
	database, err := sql.Open("sqlite3", "./auth.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	var certificateRole string
	result := database.QueryRow(fmt.Sprintf("select role from certificates where certificate  = '%s'", certificate))
	newErr := result.Scan(&certificateRole)

	if newErr != nil {
		return "none", false
	}

	return certificateRole, true
}
