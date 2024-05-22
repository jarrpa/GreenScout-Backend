package userDB

import (
	"GreenScoutBackend/constants"
	greenlogger "GreenScoutBackend/greenLogger"
	"GreenScoutBackend/rsaUtil"
	"database/sql"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var authDB *sql.DB

func InitAuthDB() {
	dbRef, dbOpenErr := sql.Open(constants.CachedConfigs.SqliteDriver, filepath.Join(constants.CachedConfigs.PathToDatabases, "auth.db"))

	authDB = dbRef

	if dbOpenErr != nil {
		greenlogger.FatalError(dbOpenErr, "Problem opening database "+filepath.Join(constants.CachedConfigs.PathToDatabases, "auth.db"))
	}
}

type LoginAttempt struct {
	Username          string
	EncryptedPassword string
}

func Authenticate(passwordEncoded []byte) (string, bool) {
	passwordPlain := rsaUtil.DecryptPassword(passwordEncoded)

	checkAgainst := make(map[string]string)

	rows, queryErr := authDB.Query("select role, password from role")

	if queryErr != nil {
		greenlogger.LogError(queryErr, "Problem in sql query SELECT role, password FROM role")
	}

	for rows.Next() {
		var role string
		var hashedWord string
		scanErr := rows.Scan(&role, &hashedWord)

		if scanErr != nil {
			greenlogger.LogError(scanErr, "Problem scanning response to sql query SELECT role, password FROM role")
		}

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
