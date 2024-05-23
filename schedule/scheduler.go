package schedule

import (
	"GreenScoutBackend/constants"
	filemanager "GreenScoutBackend/fileManager"
	greenlogger "GreenScoutBackend/greenLogger"
	"GreenScoutBackend/userDB"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
)

var scoutDB *sql.DB

func InitScoutDB() {
	dbRef, dbOpenErr := sql.Open(constants.CachedConfigs.SqliteDriver, filepath.Join("schedule", "scout.db"))

	scoutDB = dbRef

	if dbOpenErr != nil {
		greenlogger.LogErrorf(dbOpenErr, "Problem opening database %v", filepath.Join("schedule", "scout.db"))
	}
}

type ScoutRanges struct {
	Ranges [][3]int `json:"Ranges"`
}

func RetrieveSingleScouter(name string, isUUID bool) string {
	var uuid string
	if isUUID {
		uuid = name
	} else {
		uuid, _ = userDB.GetUUID(name, true)
	}

	response := scoutDB.QueryRow("select schedule from individuals where uuid = ?", uuid)

	var ranges string

	scanErr := response.Scan(&ranges)
	if scanErr != nil && !errors.Is(scanErr, sql.ErrNoRows) {
		greenlogger.LogErrorf(scanErr, "Problem scanning response %v", response)
	}

	if ranges == "" {
		return `{"Ranges":null}`
	} else {
		return ranges
	}
}

func retrieveScouterAsObject(name string, isUUID bool) ScoutRanges {

	scheduleString := RetrieveSingleScouter(name, isUUID)

	var ranges ScoutRanges

	unmarshalErr := json.Unmarshal([]byte(scheduleString), &ranges)
	if unmarshalErr != nil {
		greenlogger.LogErrorf(unmarshalErr, "Problem Unmarshalling %v", []byte(scheduleString))
	}

	return ranges
}

func AddIndividualSchedule(name string, nameIsUUID bool, ranges ScoutRanges) {

	var uuid string
	if nameIsUUID {
		uuid = name
	} else {
		uuid, _ = userDB.GetUUID(name, true)
	}

	rangeBytes, marshalErr := json.Marshal(ranges)
	if marshalErr != nil {
		greenlogger.LogErrorf(marshalErr, "Problem marshalling %v", ranges)
	}

	rangeString := string(rangeBytes)

	if userInSchedule(scoutDB, uuid) { //If doesn't exist
		cachedRanges := retrieveScouterAsObject(name, nameIsUUID)

		var newRanges ScoutRanges
		newRanges.Ranges = append(cachedRanges.Ranges, ranges.Ranges...)

		newRangeBytes, err := json.Marshal(newRanges)
		if err != nil {
			greenlogger.LogErrorf(err, "Problem marshalling %v", ranges)
		}

		rangeString = string(newRangeBytes)

		_, resultErr := scoutDB.Exec("update individuals set ranges = ? where uuid = ?", rangeString, uuid)
		if resultErr != nil {
			greenlogger.LogErrorf(resultErr, "Problem executing sql command %v with args %v", "update individuals set ranges = ? where uuid = ?", []any{rangeString, uuid})
		}

	} else {
		user := userDB.UUIDToUser(uuid)
		_, resultErr := scoutDB.Exec("insert into individuals values(?, ?, ?)", uuid, user, rangeString)
		if resultErr != nil {
			greenlogger.LogErrorf(resultErr, "Problem executing sql command %v with args %v", "insert into individuals values(?, ?, ?)", []any{uuid, user, rangeString})
		}
	}

}

func userInSchedule(database *sql.DB, uuid string) bool {
	result := database.QueryRow("select count(1) from individuals where uuid = ?", uuid)

	var resultstore int
	err := result.Scan(&resultstore)

	if err != nil {
		greenlogger.LogErrorf(err, "Problem scanning response %v", result)
	}

	return resultstore == 1
}

func WipeSchedule() {
	file, openErr := filemanager.OpenWithPermissions(filepath.Join("schedule", "schedule.json"))

	if openErr != nil {
		greenlogger.LogErrorf(openErr, "Problem opening %v", filepath.Join("schedule", "schedule.json"))
	}

	_, writeErr := file.WriteString("{}")
	if writeErr != nil {
		greenlogger.LogErrorf(writeErr, "Problem resetting %v", filepath.Join("schedule", "schedule.json"))
	}
	file.Close()
}
