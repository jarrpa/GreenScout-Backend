package userDB

type Scouter struct {
	Name  string
	Score int
}

type ModRequest struct {
	Name string
	By   int
	Mod  Modification
}

type Modification string

const (
	Increase Modification = "Increase"
	Decrease Modification = "Decrease"
	Set      Modification = "Set"
)

func ModifyUserScore(name string, alter Modification, by int) {
	uuid := GetUUID(name)

	switch alter {
	case Increase:
		userDB.Exec("update users set score = score + ? where uuid = ?", by, uuid)
	case Decrease:
		userDB.Exec("update users set score = score - ? where uuid = ?", by, uuid)
	case Set:
		userDB.Exec("update users set score = ? where uuid = ?", by, uuid)
	}
}

func GetLeaderboard() []UserInfo {
	var leaderboard []UserInfo

	resultRows, _ := userDB.Query("select uuid, username, displayname, score from users where score > 0 order by score desc")

	for resultRows.Next() {
		var uuid string
		var username string
		var displayName string
		var score int
		resultRows.Scan(&uuid, &username, &displayName, &score)

		leaderboard = append(leaderboard, UserInfo{
			Username:    username,
			DisplayName: displayName,
			Badges:      GetBadges(uuid),
			Score:       score,
		})
	}

	return leaderboard
}
