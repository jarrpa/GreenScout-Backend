package userDB

import greenlogger "GreenScoutBackend/greenLogger"

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
	uuid, _ := GetUUID(name, true)

	switch alter {
	case Increase:
		_, err := userDB.Exec("update users set score = score + ?, lifescore = lifescore + ? where uuid = ?", by, by, uuid)
		if err != nil {
			greenlogger.LogErrorf(err, "Problem executing sql query UPDATE users SET score = score + ?, lifescore = lifescore + ? WHERE uuid = ? with args: %v, %v, %v", by, by, uuid)
		}

	case Decrease:
		_, err := userDB.Exec("update users set score = score - ?, lifescore = lifescore - ? where uuid = ?", by, by, uuid)
		if err != nil {
			greenlogger.LogErrorf(err, "Problem executing sql query UPDATE users SET score = score - ?, lifescore = lifescore - ? WHERE uuid = ? with args: %v, %v, %v", by, by, uuid)
		}
	case Set: //Set will not affect life scores. Sorry!
		_, err := userDB.Exec("update users set score = ? where uuid = ?", by, uuid)
		if err != nil {
			greenlogger.LogErrorf(err, "Problem executing sql query UPDATE users SET score = ? WHERE uuid = ? with args: %v, %v", by, uuid)
		}
	}
}

func GetLeaderboard() []UserInfo {
	var leaderboard []UserInfo

	resultRows, queryErr := userDB.Query("select uuid, username, displayname, score, lifescore from users where score > 0 order by score desc")

	if queryErr != nil {
		greenlogger.LogErrorf(queryErr, "Problem executing sql query SELECT uuid, username, displayname, score, lifescore FROM users WHERE score > 0 ORDER BY score DESC")
	}

	for resultRows.Next() {
		var uuid string
		var username string
		var displayName string
		var score int
		var lifeScore int

		scanErr := resultRows.Scan(&uuid, &username, &displayName, &score, &lifeScore)

		if scanErr != nil {
			greenlogger.LogError(scanErr, "Problem scanning response to sql query SELECT uuid, username, displayname, score, lifescore FROM users WHERE score > 0 ORDER BY score DESC")
		}

		leaderboard = append(leaderboard, UserInfo{
			Username:    username,
			DisplayName: displayName,
			Badges:      GetBadges(uuid),
			Score:       score,
			LifeScore:   lifeScore,
		})
	}

	return leaderboard
}
