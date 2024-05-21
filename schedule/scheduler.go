package schedule

import (
	"GreenScoutBackend/constants"
	"GreenScoutBackend/userDB"
	"database/sql"
	"encoding/json"
)

var scoutDB, _ = sql.Open(constants.CachedConfigs.SqliteDriver, "schedule/scout.db")

type ScoutRanges struct {
	Ranges [][3]int `json:"Ranges"`
}

func RetrieveSingleScouter(name string, isUUID bool) string {
	var uuid string
	if isUUID {
		uuid = name
	} else {
		uuid = userDB.GetUUID(name)
	}

	response := scoutDB.QueryRow("select schedule from individuals where uuid = ?", uuid)

	var ranges string

	response.Scan(&ranges)

	if ranges == "" {
		return `{"Ranges":null}`
	} else {
		return ranges
	}
}

func retrieveScouterAsObject(name string, isUUID bool) ScoutRanges {

	scheduleString := RetrieveSingleScouter(name, isUUID)

	var ranges ScoutRanges

	json.Unmarshal([]byte(scheduleString), &ranges)

	return ranges
}

func AddIndividualSchedule(name string, nameIsUUID bool, ranges ScoutRanges) {

	var uuid string
	if nameIsUUID {
		uuid = name
	} else {
		uuid = userDB.GetUUID(name)
	}

	rangeBytes, _ := json.Marshal(ranges)
	rangeString := string(rangeBytes)

	if userInSchedule(scoutDB, uuid) { //If doesn't exist
		cachedRanges := retrieveScouterAsObject(name, nameIsUUID)

		var newRanges ScoutRanges
		newRanges.Ranges = append(cachedRanges.Ranges, ranges.Ranges...)

		newRangeBytes, _ := json.Marshal(newRanges)
		rangeString = string(newRangeBytes)

		scoutDB.Exec("update individuals set ranges = ? where uuid = ?", rangeString, uuid)

	} else {
		scoutDB.Exec("insert into individuals values(?, ?, ?)", uuid, userDB.UUIDToUser(uuid), rangeString)
	}

}

func userInSchedule(database *sql.DB, uuid string) bool {
	result := database.QueryRow("select count(1) from individuals where uuid = ?", uuid)

	var resultstore int
	result.Scan(&resultstore)

	return resultstore == 1
}
