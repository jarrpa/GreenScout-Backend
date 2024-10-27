package lib

// Everything + the kitchen sink

import (
	"GreenScoutBackend/constants"
	filemanager "GreenScoutBackend/fileManager"
	greenlogger "GreenScoutBackend/greenLogger"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Simple wrapper for converting bool to string for replays
func GetReplayString(isReplay bool) string {
	if isReplay {
		return "replay"
	}
	return ""
}

// Utility to check if a given array of cycles is valid for writing
func cyclesAreValid(cycles []Cycle) bool {
	return len(cycles) > 0 && cycles[0].Type != "None"
}

// Gets the number of cycles out of an array of cycles while avoiding nulls, nones, and NaNs
func GetNumCycles(cycles []Cycle) int {
	if cyclesAreValid(cycles) {
		return len(cycles)
	}

	return 0
}

// Gets the average cycle time from an array of cycles, returning N/A if cycles are invalid
func GetAvgCycleTime(cycles []Cycle) any {
	if cyclesAreValid(cycles) {
		return cycles[len(cycles)-1].Time / float64(len(cycles))
	}
	return "N/A"
}

// Gets the average cycle time from an array of cycles, returning 0 if cycles are invalid.
// Used for more number-strenuous multi scouting
func GetAvgCycleTimeExclusive(cycles []Cycle) float64 {
	if cyclesAreValid(cycles) {
		return cycles[len(cycles)-1].Time / float64(len(cycles))
	}
	return 0
}

// Calculates the total accuracy of the passed in array of cycles, returning N/A if they are invalid.
func GetCycleAccuracy(cycles []Cycle) any {
	if cyclesAreValid(cycles) {
		shotsMade := 0
		for _, cycle := range cycles {
			if cycle.Success {
				shotsMade++
			}
		}
		return (float64(shotsMade) / float64(len(cycles))) * 100
	}
	return "N/A"
}

// Calculates the tendencies of amp, speaker, shuttling, and distance shooting from an array of cycles.
// Returns: Tendency to AMP, SPEAKER, DISTANCE, SHUTTLE
func GetCycleTendencies(cycles []Cycle) (float64, float64, float64, float64) {
	if len(cycles) < 1 {
		return 0, 0, 0, 0
	}

	var numAmps float64
	var numSpeakers float64
	var numShuttles float64
	var numDistances float64

	numCycles := len(cycles)

	for _, cycle := range cycles {
		switch cycle.Type {
		case "Amp":
			numAmps++
		case "Speaker":
			numSpeakers++
		case "Shuttle":
			numShuttles++
		case "Distance":
			numDistances++
		}
	}

	return numAmps / float64(numCycles),
		numSpeakers / float64(numCycles),
		numDistances / float64(numCycles),
		numShuttles / float64(numCycles)
}

// Calculates the accuracies of amp, speaker, shuttling, and distance shooting from an array of cycles, returning N/A for any that had 0 attempts.
// Returns: Accuracy of AMP, SPEAKER, DISTANCE, SHUTTLE
func GetCycleAccuracies(cycles []Cycle) (any, any, any, any) {
	if cyclesAreValid(cycles) {
		ampsAttempted, ampsMade := 0, 0
		speakersAttempted, speakersMade := 0, 0
		distancesAttempted, distancesMade := 0, 0
		shuttlesAttempted, shuttlesMade := 0, 0

		for _, cycle := range cycles {
			switch cycle.Type {
			case "Amp":
				{
					ampsAttempted++
					if cycle.Success {
						ampsMade++
					}
				}
			case "Speaker":
				{
					speakersAttempted++
					if cycle.Success {
						speakersMade++
					}
				}
			case "Distance":
				{
					distancesAttempted++
					if cycle.Success {
						distancesMade++
					}
				}
			case "Shuttle":
				{
					shuttlesAttempted++
					if cycle.Success {
						shuttlesMade++
					}
				}
			}
		}

		var ampAccuracy any
		var speakerAccuracy any
		var distanceAccuracy any
		var shuttleAccuracy any

		if ampsAttempted == 0 {
			ampAccuracy = "N/A"
		} else {
			ampAccuracy = (float64(ampsMade) / float64(ampsAttempted)) * 100
		}

		if speakersAttempted == 0 {
			speakerAccuracy = "N/A"
		} else {
			speakerAccuracy = (float64(speakersMade) / float64(speakersAttempted)) * 100
		}

		if distancesAttempted == 0 {
			distanceAccuracy = "N/A"
		} else {
			distanceAccuracy = (float64(distancesMade) / float64(distancesAttempted)) * 100
		}

		if shuttlesAttempted == 0 {
			shuttleAccuracy = "N/A"
		} else {
			shuttleAccuracy = (float64(shuttlesMade) / float64(shuttlesAttempted)) * 100
		}

		return ampAccuracy, speakerAccuracy, distanceAccuracy, shuttleAccuracy
	}
	return "N/A", "N/A", "N/A", "N/A"
}

// Gets the accuracy of a robot during an autonomous period, returning N/A if 0 attempts were made
func GetAutoAccuracy(auto AutoData) any {
	attempts := auto.Scores + auto.Misses

	if attempts == 0 {
		return "N/A"
	}
	return (float64(auto.Scores) / float64(attempts)) * 100
}

// Compiles Losing track, DCs, penalties, and notes into one string of notes
func CompileNotes(team TeamData) string {
	var finalNote string = ""
	if team.Misc.LostTrack {
		finalNote += "LOST TRACK; "
	}

	if team.Misc.DC || team.Misc.Disabled {
		finalNote += "DISCONNECTED; "
	}

	if len(team.Penalties) > 0 {
		finalNote += "PENALTIES= " + strings.Join(team.Penalties, ",") + "; "
	}

	finalNote += team.Notes
	return finalNote
}

// Compiles Losing track, DCs, and notes into one string of notes.
// Used for multi-scouting only
func CompileNotes2(match MultiMatch, teams []TeamData) string {
	var finalNote string = ""
	var lostTrack bool = false
	var DC bool = false

	for _, entry := range teams {
		if entry.Misc.LostTrack {
			lostTrack = true
		}

		if entry.Misc.DC || entry.Misc.Disabled {
			DC = true
		}
	}

	if lostTrack {
		finalNote += "LOST TRACK; "
	}

	if DC {
		finalNote += "DISCONNECTED; "
	}

	finalNote += strings.Join(match.Notes, "; ")
	return finalNote
}

// Returns if a file exists in Teamlists matching the passed in event key
func CheckForTeamLists(eventKey string) bool {
	_, err := os.Open(filepath.Join(constants.CachedConfigs.TeamListsDirectory, eventKey))

	return err == nil
}

// Writes the teams attending an event to the matching file in TeamLists
func WriteTeamsToFile(configs constants.GeneralConfigs) {
	runnable := exec.Command(configs.PythonDriver, "getTeamList.py", configs.TBAKey, configs.EventKey, configs.TeamListsDirectory)

	_, err := runnable.Output()

	if err != nil && !strings.Contains(err.Error(), "exit status 1") {
		greenlogger.LogErrorf(err, "Error executing command %v %v %v %v %v", configs.PythonDriver, "getTeamlist.py", configs.TBAKey, configs.EventKey, configs.TeamListsDirectory)
	}
}

// Reads the Teams from teamlists and stores them in memory
func StoreTeams() {
	pathToCurrEvent := filepath.Join(constants.CachedConfigs.TeamListsDirectory, GetCurrentEvent())

	file, err := os.Open(pathToCurrEvent)

	if err != nil {
		greenlogger.LogErrorf(err, "Error opening %v", pathToCurrEvent)
	}

	resultBytes, readErr := io.ReadAll(file)
	resultStr := strings.Split(string(resultBytes), "\n")[1:]

	var resultInts []int
	for _, result := range resultStr {
		if result != "" {
			parsed, err := strconv.ParseInt(result, 10, 64)
			if err != nil {
				greenlogger.LogErrorf(err, "Error parsing %v as int", result)
			}
			resultInts = append(resultInts, int(parsed))
		}
	}

	if readErr != nil {
		greenlogger.LogErrorf(readErr, "Error reading %v", pathToCurrEvent)
	}

	constants.Teams = resultInts
}

// Writes the schedule of an event to schedule/schedule.json
func WriteScheduleToFile(configs constants.GeneralConfigs) {
	runnable := exec.Command(configs.PythonDriver, "getSchedule.py", configs.TBAKey, configs.EventKey, configs.RuntimeDirectory)

	_, err := runnable.Output()

	if err != nil && !strings.Contains(err.Error(), "exit status 1") {
		greenlogger.LogErrorf(err, "Error executing command %v %v %v", configs.PythonDriver, "getSchedule.py", configs.EventKey)
	}
}

// Writes all events for the current year to events.json
func WriteEventsToFile(configs constants.GeneralConfigs) {
	runnable := exec.Command(configs.PythonDriver, "getAllEvents.py", configs.TBAKey)

	out, err := runnable.Output()

	if err != nil && !strings.Contains(err.Error(), "exit status 1") {
		greenlogger.LogErrorf(err, "Error executing command %v %v %v", configs.PythonDriver, "getAllEvents.py", configs.TBAKey)
	}

	if strings.Contains(string(out), "ERR") {
		greenlogger.LogMessagef("Error executing command %v %v %v; Investigate in python", configs.PythonDriver, "getAllEvents.py", configs.TBAKey)

	}
}

// Calculates the string from a SpeakerPositions object
func GetSpeakerPosAsString(positions SpeakerPositions) string {
	if positions.Sides && positions.Middle {
		return "BOTH"
	}

	if positions.Middle {
		return "MIDDLE"
	} else if positions.Sides {
		return "SIDES"
	} else {
		return "NONE"
	}
}

// Calculates the string from a PickupLocations object
func GetPickupLocations(locations PickupLocations) string {
	if locations.Ground && locations.Source {
		return "BOTH"
	}

	if locations.Ground {
		return "GROUND"
	} else if locations.Source {
		return "SOURCE"
	} else {
		return "NONE"
	}
}

// Calculates the string from data pertaining to a driverstation
func GetDSString(isBlue bool, number uint) string {
	var builder string = ""

	if isBlue {
		builder += "blue"
	} else {
		builder += "red"
	}

	builder += fmt.Sprint(number)

	return builder
}

// turns the driverstation string into an int 1-5 representing its absolute number
func GetDSOffset(ds string) int {
	switch chooser := ds; chooser {
	case "red1":
		return 0
	case "red2":
		return 1
	case "red3":
		return 2
	case "blue1":
		return 3
	case "blue2":
		return 4
	case "blue3":
		return 5
	}

	return 0
}

// Gets the row an entry will write to from its Teamdata object
func GetRow(team TeamData) int {
	startRow := 2 + (team.Match.Number-1)*6
	dsString := GetDSString(team.DriverStation.IsBlue, uint(team.DriverStation.Number))
	dsOffset := GetDSOffset(dsString)

	startRow += uint(dsOffset)

	return int(startRow)
}

// Gets the row a pit scouting data should write to
func GetPitRow(team int) int {
	return slices.Index(constants.Teams, team) + 1
}

// Getter for the current event key
func GetCurrentEvent() string {
	return constants.CachedConfigs.EventKey
}

// Compares two slices for equality
func CompareSplits(first []string, second []string) bool {
	if len(first) != len(second) {
		return false
	}

	for i, element := range first {
		if element != second[i] {
			return false
		}
	}

	return true
}

// Gets all files matching the passed in pattern; Used to find all files of one entry when multi-scouting
func GetAllMatching(checkAgainst string) []string {
	var results []string
	splitAgainst := strings.Split(checkAgainst, "_")

	writtenJson, err := os.ReadDir(constants.JsonWrittenDirectory)

	if err != nil {
		greenlogger.LogErrorf(err, "Error reading directory %v", constants.JsonWrittenDirectory)
		return results
	}

	if len(writtenJson) > 0 {
		for _, jsonFile := range writtenJson {
			splitFile := strings.Split(jsonFile.Name(), "_")

			if len(splitFile) < 4 {
				continue
			}

			if CompareSplits(splitAgainst[:3], splitFile[:3]) {
				results = append(results, jsonFile.Name())
			}
		}
	}
	return results
}

// Gets the number of matches from schedule.json
func GetNumMatches() int {
	var result map[int]map[string][]int // pain

	jsonPath := filepath.Join(constants.CachedConfigs.RuntimeDirectory, "schedule.json")
	file, err := os.Open(jsonPath)

	if err != nil {
		greenlogger.LogErrorf(err, "Error opening %v", jsonPath)
		return len(result)
	}

	decodeErr := json.NewDecoder(file).Decode(&result)
	if decodeErr != nil {
		greenlogger.LogErrorf(err, "Error Decoding %v", jsonPath)
		return len(result)
	}

	return len(result)
}

// Moves a file from an original path to a new one, returning wether or not it was successful
func MoveFile(originalPath string, newPath string) bool {
	oldLoc, openErr := os.Open(originalPath)

	if openErr != nil {
		greenlogger.LogErrorf(openErr, "Error opening %v", originalPath)
		return false
	}

	newLoc, openErr := filemanager.OpenWithPermissions(newPath)
	if openErr != nil {
		greenlogger.LogErrorf(openErr, "Error creating %v", newPath)
		return false
	}

	defer newLoc.Close()

	_, copyErr := io.Copy(newLoc, oldLoc)

	if copyErr != nil {
		greenlogger.LogErrorf(copyErr, "Error copying %v to %v", originalPath, newPath)
		return false
	}

	if closeErr := oldLoc.Close(); closeErr != nil { //This is NOT a cause of returning false
		greenlogger.LogError(copyErr, "Error closing "+originalPath)
	}

	if removeErr := os.Remove(originalPath); removeErr != nil {
		greenlogger.LogError(removeErr, "Error removing "+originalPath)
		return false
	}

	return true
}

// Gets the distance a robot can shoot from, returning N/A if it has no ability to do so.
func GetDistance(data PitScoutingData) any {
	if data.Distance.Can {
		return int(data.Distance.Distance)
	}

	return "N/A"
}

// Gets the time a robot takes to climb, returning N/A if it doesn't do so in endgame.
func GetClimbTime(data PitScoutingData) any {
	if data.EndgameBehavior == "Climb" {
		return data.ClimbTime
	}

	return "N/A"
}
