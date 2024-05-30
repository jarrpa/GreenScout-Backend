package userDB

import (
	greenlogger "GreenScoutBackend/greenLogger"
	"database/sql"
	"errors"
)

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
	checkAndUpdateHighScore(uuid)
}

func checkAndUpdateHighScore(uuid string) {
	result := userDB.QueryRow("select score > highscore from users where uuid = ?", uuid)

	var larger bool

	scanErr := result.Scan(&larger)
	if scanErr != nil && !errors.Is(scanErr, sql.ErrNoRows) {
		greenlogger.LogErrorf(scanErr, "Problem scanning response %v", result)
	}

	if larger {
		_, execErr := userDB.Exec("update users set highscore = score where uuid = ?", uuid)
		if execErr != nil {
			greenlogger.LogErrorf(execErr, "Problem executing sql query UPDATE users SET highscore = score WHERE uuid = ? with arg: %v", uuid)
		}
	}
}

func GetLeaderboard() []UserInfo {
	var leaderboard []UserInfo

	resultRows, queryErr := userDB.Query("select uuid, username, displayname, score, lifescore, highscore from users where score > 0 order by score desc")

	if queryErr != nil {
		greenlogger.LogErrorf(queryErr, "Problem executing sql query SELECT uuid, username, displayname, score, lifescore FROM users WHERE score > 0 ORDER BY score DESC")
	}

	for resultRows.Next() {
		var uuid string
		var username string
		var displayName string
		var score int
		var lifeScore int
		var highScore int

		scanErr := resultRows.Scan(&uuid, &username, &displayName, &score, &lifeScore, &highScore)

		if scanErr != nil {
			greenlogger.LogError(scanErr, "Problem scanning response to sql query SELECT uuid, username, displayname, score, lifescore, highscore FROM users WHERE score > 0 ORDER BY score DESC")
		}

		leaderboard = append(leaderboard, UserInfo{
			Username:    username,
			DisplayName: displayName,
			Badges:      GetBadges(uuid),
			Score:       score,
			LifeScore:   lifeScore,
			HighScore:   highScore,
		})
	}

	return leaderboard
}
