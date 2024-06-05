package lib

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
	"strconv"
	"strings"
)

func GetReplayString(isReplay bool) string {
	if isReplay {
		return "replay"
	}
	return ""
}

func cyclesAreValid(cycles []Cycle) bool {
	return len(cycles) > 0 && cycles[0].Type != "None"
}

func GetNumCycles(cycles []Cycle) int {
	if cyclesAreValid(cycles) {
		return len(cycles)
	}

	return 0
}

func GetAvgCycleTime(cycles []Cycle) any {
	if cyclesAreValid(cycles) {
		return cycles[len(cycles)-1].Time / float64(len(cycles))
	}
	return "N/A"
}

func GetAvgCycleTimeExclusive(cycles []Cycle) float64 {
	if cyclesAreValid(cycles) {
		return cycles[len(cycles)-1].Time / float64(len(cycles))
	}
	return 0
}

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

type CycleData struct {
	Amp      float64
	Speaker  float64
	Distance float64
	Shuttle  float64
}

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

func GetCycleAccuracies(cycles []Cycle) (any, any, any, any) { //These are any and not floats because they can be N/A
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

func GetAutoAccuracy(auto AutoData) any {
	attempts := auto.Scores + auto.Misses

	if attempts == 0 {
		return "N/A"
	}
	return (float64(auto.Scores) / float64(attempts)) * 100
}

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

func CheckForTeamLists(eventKey string) bool {
	_, err := os.Open(filepath.Join("TeamLists", "$"+eventKey))

	return err == nil
}

func WriteTeamsToFile(apiKey string, eventKey string) {
	runnable := exec.Command(constants.CachedConfigs.PythonDriver, "getTeamList.py", apiKey, eventKey)

	_, err := runnable.Output()

	if err != nil && !strings.Contains(err.Error(), "exit status 1") {
		greenlogger.LogErrorf(err, "Error executing command %v %v %v %v", constants.CachedConfigs.PythonDriver, "getTeamlist.py", apiKey, eventKey)
	}
}

func StoreTeams() {
	pathToCurrEvent := filepath.Join("TeamLists", "$"+GetCurrentEvent())

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

func WriteScheduleToFile(key string) {
	runnable := exec.Command(constants.CachedConfigs.PythonDriver, "getSchedule.py", constants.CachedConfigs.TBAKey, key)

	_, err := runnable.Output()

	if err != nil && !strings.Contains(err.Error(), "exit status 1") {
		greenlogger.LogErrorf(err, "Error executing command %v %v %v", constants.CachedConfigs.PythonDriver, "getSchedule.py", key)
	}
}

func WriteEventsToFile() {
	runnable := exec.Command(constants.CachedConfigs.PythonDriver, "getAllEvents.py", constants.CachedConfigs.TBAKey)

	out, err := runnable.Output()

	if err != nil && !strings.Contains(err.Error(), "exit status 1") {
		greenlogger.LogErrorf(err, "Error executing command %v %v %v", constants.CachedConfigs.PythonDriver, "getAllEvents.py", constants.CachedConfigs.TBAKey)
	}

	if strings.Contains(string(out), "ERR") {
		greenlogger.LogMessagef("Error executing command %v %v %v; Investigate in python", constants.CachedConfigs.PythonDriver, "getAllEvents.py", constants.CachedConfigs.TBAKey)

	}
}

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

func GetRow(team TeamData) int {
	startRow := 2 + (team.Match.Number-1)*6
	dsString := GetDSString(team.DriverStation.IsBlue, uint(team.DriverStation.Number))
	dsOffset := GetDSOffset(dsString)

	startRow += uint(dsOffset)

	return int(startRow)
}

func GetPitRow(team int) {

}

func GetCurrentEvent() string {
	return constants.CachedConfigs.EventKey
}

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

func GetAllMatching(checkAgainst string) []string {
	var results []string
	splitAgainst := strings.Split(checkAgainst, "_")

	writtenJson, err := os.ReadDir(filepath.Join("InputtedJson", "Written"))

	if err != nil {
		greenlogger.LogErrorf(err, "Error reading directory %v", filepath.Join("InputtedJson", "Written"))
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

func GetNumMatches() int {
	var result map[int]map[string][]int

	file, err := os.Open(filepath.Join("schedule", "schedule.json"))

	if err != nil {
		greenlogger.LogErrorf(err, "Error opening %v", filepath.Join("schedule", "schedule.json"))
		return len(result)
	}

	decodeErr := json.NewDecoder(file).Decode(&result)
	if decodeErr != nil {
		greenlogger.LogErrorf(err, "Error Decoding %v", filepath.Join("schedule", "schedule.json"))
		return len(result)
	}

	return len(result)
}

// Bool is success
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
		greenlogger.ElogError(copyErr, "Error closing "+originalPath)
	}

	if removeErr := os.Remove(originalPath); removeErr != nil {
		greenlogger.LogError(removeErr, "Error removing "+originalPath)
		return false
	}

	return true
}

func GetDistance(data PitScoutingData) any {
	if data.Distance.Can {
		return int(data.Distance.Distance)
	}

	return "N/A"
}

func GetClimbTime(data PitScoutingData) any {
	if data.EndgameBehavior == "Climb" {
		return data.ClimbTime
	}

	return "N/A"
}
