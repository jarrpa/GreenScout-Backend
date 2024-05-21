package userDB

import (
	"GreenScoutBackend/constants"
	"GreenScoutBackend/rsaUtil"
	"database/sql"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var authDB, _ = sql.Open("sqlite3", filepath.Join(constants.CachedConfigs.PathToDatabases, "auth.db"))

type LoginAttempt struct {
	Username          string
	EncryptedPassword string
}

func Authenticate(passwordEncoded []byte) (string, bool) {
	passwordPlain := rsaUtil.DecryptPassword(passwordEncoded)

	checkAgainst := make(map[string]string)

	rows, _ := authDB.Query("select role, password from role")

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

func comparePasswordBCrypt(plainPassword string, encodedPassword string) bool {
	err := bcrypt.CompareHashAndPassword(
		[]byte(encodedPassword), []byte(plainPassword))

	return err == nil
}
