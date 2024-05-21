package userDB

import (
	"GreenScoutBackend/constants"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

var userDB, _ = sql.Open("sqlite3", filepath.Join(constants.CachedConfigs.PathToDatabases, "users.db"))

func GenTables() {
	_, err := userDB.Exec("create table users(uuid string not null primary key, username string, displayname string, certificate string, badges string[], score integer, pfp string)")

	if err != nil {
		panic(err)
	}
}

func NewUser(username string, uuid string) {
	badgeBytes, _ := json.Marshal(emptyBadges())

	_, err := userDB.Exec("insert into users values(?,?,?,?,?,?,?)", uuid, username, username, nil, string(badgeBytes), 0, filepath.Join("pfp", "pictures", "Default_pfp.png"))
	if err != nil {
		print(err.Error())
	}
}

func userExists(username string) bool {
	result := userDB.QueryRow("select count(1) from users where username = ?", username)

	var resultstore int
	result.Scan(&resultstore)

	return resultstore == 1
}

func GetUUID(username string) string {
	if !userExists(username) {
		NewUser(username, "") //Empty UUID for assignment later
	}

	var userId string
	result := userDB.QueryRow("select uuid from users where username = ?", username)
	result.Scan(&userId)

	if userId == "" {
		newId := uuid.New()
		_, err := userDB.Exec("update users set uuid = ? where username = ?", newId, username)

		if err != nil {
			println(err.Error())
		}

		userId = newId.String()
	}
	return userId
}

func UUIDToUser(uuid string) string {
	result := userDB.QueryRow(fmt.Sprintf("select username from users where uuid = '%s'", uuid))
	var resultStore string
	result.Scan(&resultStore)

	return resultStore
}

type User struct {
	Name string
	UUID string
}

func GetAllUsers() []User {
	result, _ := userDB.Query("select username, uuid from users")

	var users []User

	for result.Next() {
		var name string
		var uuid string
		result.Scan(&name, &uuid)
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

type UserInfo struct {
	Username    string
	DisplayName string
	Badges      []Badge
	Score       int
}

func GetUserInfo(username string) UserInfo {
	uuid := GetUUID(username)

	userInfo := UserInfo{
		Username:    username,
		DisplayName: GetDisplayName(uuid),
		Badges:      GetBadges(uuid),
		Score:       getScore(uuid),
	}
	return userInfo
}

func GetDisplayName(uuid string) string {
	var displayName string
	response := userDB.QueryRow("select displayname from users where uuid = ?", uuid)
	response.Scan(&displayName)
	return displayName
}

func GetBadges(uuid string) []Badge {
	var Badges []Badge
	var BadgesMarshalled string
	response := userDB.QueryRow("select badges from users where uuid = ?", uuid)
	response.Scan(&BadgesMarshalled)
	// i am aware of how awful converting []byte -> string -> []byte is but i've had problems storing byte arrays with sqlite. postgres doesn't have this problem but what high schooler is learning postgres
	json.Unmarshal([]byte(BadgesMarshalled), &Badges)
	return Badges
}

func emptyBadges() []Badge {
	return []Badge{}
}

func SetDisplayName(username string, displayName string) {
	uuid := GetUUID(username)

	userDB.Exec("update users set displayname = ? where uuid = ?", displayName, uuid)
}

func AddBadge(uuid string, badge Badge) {
	existingBadges := GetBadges(uuid)

	existingBadges = append(existingBadges, badge)

	badgesBytes, _ := json.Marshal(existingBadges)

	userDB.Exec("update users set badges = ? where uuid = ?", string(badgesBytes), uuid)
}

func getScore(uuid string) int {
	var score int
	response := userDB.QueryRow("select score from users where uuid = ?", uuid)
	response.Scan(&score)
	return score
}
