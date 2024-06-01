package userDB

import (
	"GreenScoutBackend/constants"
	greenlogger "GreenScoutBackend/greenLogger"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"slices"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

var userDB *sql.DB

func InitUserDB() {
	dbRef, dbOpenErr := sql.Open(constants.CachedConfigs.SqliteDriver, filepath.Join(constants.CachedConfigs.PathToDatabases, "users.db"))

	userDB = dbRef

	if dbOpenErr != nil {
		greenlogger.FatalError(dbOpenErr, "Problem opening database "+filepath.Join(constants.CachedConfigs.PathToDatabases, "users.db"))
	}
}

func NewUser(username string, uuid string) {
	badgeBytes, marshalError := json.Marshal(emptyBadges())

	if marshalError != nil {
		greenlogger.LogError(marshalError, "Problem marshalling empty badge JSON")
	}

	//Lifescore & highscore aren't changed here because they have default values. The only reason the others don't is that sqlite doesn't let you alter column default values and I don't feel like deleting and remaking every column
	_, err := userDB.Exec("insert into users values(?,?,?,?,?,?,?)", uuid, username, username, nil, string(badgeBytes), 0, filepath.Join("pfp", "pictures", "Default_pfp.png"))

	if err != nil {
		greenlogger.LogErrorf(err, "Problem creating new user with args: %v, %v, %v, %v, %v, %v, %v", uuid, username, username, "nil", badgeBytes, 0, filepath.Join("pfp", "pictures", "Default_pfp.png"))
	}
}

func userExists(username string) bool {
	result := userDB.QueryRow("select count(1) from users where username = ?", username)

	var resultstore int
	scanErr := result.Scan(&resultstore)

	if scanErr != nil {
		greenlogger.LogError(scanErr, "Problem scanning response to sql query SELECT COUNT(1) FROM users WHERE username = ? with arg: "+username)
	}

	return resultstore == 1
}

func GetUUID(username string, createIfNot bool) (string, bool) {
	userExists := userExists(username)

	if (!createIfNot) && !userExists {
		return "", false
	}

	if !userExists {
		NewUser(username, "") //Empty UUID for assignment later
	}

	var userId string
	result := userDB.QueryRow("select uuid from users where username = ?", username)
	scanErr := result.Scan(&userId)
	if scanErr != nil {
		greenlogger.LogError(scanErr, "Problem scanning response to sql query SELECT uuid FROM users WHERE username = ? with arg: "+username)
	}

	if userId == "" {
		newId := uuid.New()
		_, err := userDB.Exec("update users set uuid = ? where username = ?", newId, username)

		if err != nil {
			greenlogger.LogErrorf(err, "Problem executing sql query UPDATE users SET uuid = ? WHERE username = ? with args: %v, %v", newId, username)
		}

		userId = newId.String()
	}
	return userId, true
}

func UUIDToUser(uuid string) string {
	result := userDB.QueryRow("select username from users where uuid = ?", uuid)
	var resultStore string
	err := result.Scan(&resultStore)

	if err != nil {
		greenlogger.LogErrorf(err, "Problem scanning results of sql query SELECT username FROM users WHERE uuid = ? with arg: %v", uuid)
	}

	return resultStore
}

type User struct {
	Name string
	UUID string
}

func GetAllUsers() []User {
	result, err := userDB.Query("select username, uuid from users")

	if err != nil {
		greenlogger.LogError(err, "Problem executing sql query SELECT username, uuid FROM users")
	}

	var users []User

	for result.Next() {
		var name string
		var uuid string
		scanErr := result.Scan(&name, &uuid)

		if scanErr != nil {
			greenlogger.LogError(scanErr, "Problem scanning results of sql query SELECT username, uuid FROM users")
		}

		if name != "" {
			users = append(users, User{Name: name, UUID: uuid})
		}
	}

	return users
}

type Badge struct {
	ID          string
	Description string
}

type Accolade string

type UserInfo struct {
	Username    string
	DisplayName string
	Accolades   []Accolade //Leaderboard-invisible
	Badges      []Badge    //Leaderboard-visible
	Score       int
	LifeScore   int
	HighScore   int
	Pfp         string
}

func GetUserInfo(username string) UserInfo {
	uuid, exists := GetUUID(username, false)

	var displayName string
	var accolades []Accolade
	var badges []Badge
	var score int
	var lifeScore int
	var highscore int

	var pfp string

	if exists { // This could 100% be made more efficient, but not my problem!
		displayName = GetDisplayName(uuid)
		badges = GetBadges(uuid)
		accolades = GetAccolades(uuid)
		score = getScore(uuid)
		lifeScore = getLifeScore(uuid)
		highscore = getHighScore(uuid)
		pfp = getPfp(uuid)
	} else {
		displayName = "User does not exist"
		badges = emptyBadges()
		accolades = emptyAccolades()
		score = -1
		lifeScore = -1
		highscore = -1
		pfp = constants.DefaultPfpPath
	}

	userInfo := UserInfo{
		Username:    username,
		DisplayName: displayName,
		Badges:      badges,
		Accolades:   accolades,
		Score:       score,
		LifeScore:   lifeScore,
		HighScore:   highscore,
		Pfp:         pfp,
	}
	return userInfo
}

func GetDisplayName(uuid string) string {
	var displayName string
	response := userDB.QueryRow("select displayname from users where uuid = ?", uuid)
	scanErr := response.Scan(&displayName)
	if scanErr != nil {
		greenlogger.LogError(scanErr, "Problem scanning results of sql query SELECT displayname FROM users WHERE uuid = ? with arg: "+uuid)
	}

	return displayName
}

func GetBadges(uuid string) []Badge {
	var Badges []Badge
	var BadgesMarshalled string
	response := userDB.QueryRow("select badges from users where uuid = ?", uuid)
	scanErr := response.Scan(&BadgesMarshalled)
	if scanErr != nil {
		greenlogger.LogError(scanErr, "Problem scanning results of sql query SELECT badges FROM users WHERE uuid = ? with arg: "+uuid)
	}
	// i am aware of how awful converting []byte -> string -> []byte is but i've had problems storing byte arrays with sqlite. postgres doesn't have this problem but what high schooler is learning postgres
	unmarshalErr := json.Unmarshal([]byte(BadgesMarshalled), &Badges)
	if unmarshalErr != nil {
		greenlogger.LogErrorf(unmarshalErr, "Problem unmarshalling %v", BadgesMarshalled)
	}

	return Badges
}

func emptyBadges() []Badge {
	return []Badge{}
}

func emptyAccolades() []Accolade {
	return []Accolade{}
}

func GetAccolades(uuid string) []Accolade {
	var Accolades []Accolade
	var AccoladesMarshalled string
	response := userDB.QueryRow("select accolades from users where uuid = ?", uuid)
	scanErr := response.Scan(&AccoladesMarshalled)
	if scanErr != nil {
		greenlogger.LogError(scanErr, "Problem scanning results of sql query SELECT accolades FROM users WHERE uuid = ? with arg: "+uuid)
	}
	// i am aware of how awful converting []byte -> string -> []byte is but i've had problems storing byte arrays with sqlite. postgres doesn't have this problem but what high schooler is learning postgres
	unmarshalErr := json.Unmarshal([]byte(AccoladesMarshalled), &Accolades)
	if unmarshalErr != nil {
		greenlogger.LogErrorf(unmarshalErr, "Problem unmarshalling %v", AccoladesMarshalled)
	}

	return Accolades
}

func SetDisplayName(username string, displayName string) {
	uuid, _ := GetUUID(username, true)

	_, execErr := userDB.Exec("update users set displayname = ? where uuid = ?", displayName, uuid)

	if execErr != nil {
		greenlogger.LogErrorf(execErr, "Problem executing sql query UPDATE users SET displayname = ? WHERE uuid = ? with args: %v, %v", displayName, uuid)
	}
}

func AddAccolade(uuid string, accolade Accolade) {
	existingAccolades := GetAccolades(uuid)

	if !slices.Contains(existingAccolades, accolade) {
		existingAccolades = append(existingAccolades, accolade)
	}

	accBytes, marshalErr := json.Marshal(existingAccolades)
	if marshalErr != nil {
		greenlogger.LogErrorf(marshalErr, "Problem marshalling %v", existingAccolades)
	}

	_, execErr := userDB.Exec("update users set accolades = ? where uuid = ?", string(accBytes), uuid)
	if execErr != nil {
		greenlogger.LogErrorf(execErr, "Problem executing sql query UPDATE users SET accolades = ? WHERE uuid = ? with args: %v, %v", accBytes, uuid)
	}
}

func AddBadge(uuid string, badge Badge) {
	existingBadges := GetBadges(uuid)

	var toAppend = true
	for i := range existingBadges {
		if existingBadges[i].ID == badge.ID {
			toAppend = false

			if existingBadges[i].Description != badge.Description {
				existingBadges[i].Description = badge.Description
			}

			break
		}
	}

	if toAppend {
		existingBadges = append(existingBadges, badge)
	}

	badgesBytes, marshalErr := json.Marshal(existingBadges)
	if marshalErr != nil {
		greenlogger.LogErrorf(marshalErr, "Problem marshalling %v", existingBadges)
	}

	_, execErr := userDB.Exec("update users set badges = ? where uuid = ?", string(badgesBytes), uuid)
	if execErr != nil {
		greenlogger.LogErrorf(execErr, "Problem executing sql query UPDATE users SET badges = ? WHERE uuid = ? with args: %v, %v", badgesBytes, uuid)
	}

}

func getScore(uuid string) int {
	var score int
	response := userDB.QueryRow("select score from users where uuid = ?", uuid)
	scanErr := response.Scan(&score)
	if scanErr != nil {
		greenlogger.LogError(scanErr, "Problem scanning response to sql query SELECT score FROM users WHERE uuid = ? with arg: "+uuid)
	}

	return score
}

func getLifeScore(uuid string) int {
	var score int
	response := userDB.QueryRow("select lifescore from users where uuid = ?", uuid)
	scanErr := response.Scan(&score)
	if scanErr != nil {
		greenlogger.LogError(scanErr, "Problem scanning response to sql query SELECT lifescore FROM users WHERE uuid = ? with arg: "+uuid)
	}

	return score
}

func getHighScore(uuid string) int {
	var highscore int
	response := userDB.QueryRow("select highscore from users where uuid = ?", uuid)
	scanErr := response.Scan(&highscore)
	if scanErr != nil {
		greenlogger.LogError(scanErr, "Problem scanning response to sql query SELECT highscore FROM users WHERE uuid = ? with arg: "+uuid)
	}

	return highscore
}

func getPfp(uuid string) string {
	var pfp string
	response := userDB.QueryRow("select pfp from users where uuid = ?", uuid)
	scanErr := response.Scan(&pfp)
	if scanErr != nil {
		greenlogger.LogError(scanErr, "Problem scanning response to sql query SELECT pfp FROM users WHERE uuid = ? with arg: "+uuid)
	}

	return pfp
}

func SetPfp(username string, pfp string) {
	uuid, _ := GetUUID(username, true)

	_, execErr := userDB.Exec("update users set pfp = ? where uuid = ?", pfp, uuid)

	if execErr != nil {
		greenlogger.LogErrorf(execErr, "Problem executing sql query UPDATE users SET pfp = ? WHERE uuid = ? with args: %v, %v", pfp, uuid)
	}
}
