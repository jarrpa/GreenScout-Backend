package userDB

// Utilities for handling auth.db

import (
	"GreenScoutBackend/constants"
	greenlogger "GreenScoutBackend/greenLogger"
	"GreenScoutBackend/rsaUtil"
	"database/sql"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// The auth.db database reference
var authDB *sql.DB

// Initializes auth.db and stores it to memory
func InitAuthDB() {
	dbRef, dbOpenErr := sql.Open(constants.CachedConfigs.SqliteDriver, filepath.Join(constants.CachedConfigs.PathToDatabases, "auth.db"))

	authDB = dbRef

	if dbOpenErr != nil {
		greenlogger.FatalError(dbOpenErr, "Problem opening database "+filepath.Join(constants.CachedConfigs.PathToDatabases, "auth.db"))
	}
}

// An attempt to log in
type LoginAttempt struct {
	Username          string
	EncryptedPassword string
}

// Authenticates the password through the database, returning the role it turned out to be and if it authenticated.
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

// A wrapper for comparing an unhashed and hashed password through bcrypt
func comparePasswordBCrypt(plainPassword string, encodedPassword string) bool {
	err := bcrypt.CompareHashAndPassword(
		[]byte(encodedPassword), []byte(plainPassword))

	return err == nil
}
