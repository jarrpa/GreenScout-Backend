package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var teamColumns = make(map[int]string)
var teamListInterface [][]interface{}
var maxColumn int

var CachedKey *string

func GetReplayString(isReplay bool) string {
	if isReplay {
		return "replay"
	}
	return ""
}

func cyclesAreValid(cycles []Cycle) bool {
	return cycles[0].Type != "None"
}

func GetNumCycles(cycles []Cycle) any {
	if cyclesAreValid(cycles) {
		return len(cycles)
	}

	return "N/A"
}

func GetAvgCycleTime(cycles []Cycle) any {
	if cyclesAreValid(cycles) {
		var sum float64

		for i := 0; i < len(cycles); i++ {
			sum += cycles[i].Time
		}

		return sum / float64(len(cycles))
	}
	return "N/A"
}

func GetAccuracy(data DistanceShotData) float64 {
	attempts := data.Misses + data.Scores

	if attempts == 0 {
		return 0
	}

	return (float64(data.Scores)) / float64(attempts)
}

func GetCyclePercents(team TeamData) [2]float64 {
	var numAmps float64
	var numSpeakers float64

	cycles := len(team.Cycles)

	for _, cycle := range team.Cycles {
		if cycle.Type == "Amp" {
			numAmps++
		} else {
			numSpeakers++
		}
	}

	return [2]float64{numAmps / float64(cycles), numSpeakers / float64(cycles)}
}

func GetCachedEventKey() string {
	if CachedKey == nil {
		file, _ := os.Open("config/GreenScoutConfig.json")
		defer file.Close()

		var configs EventConfig
		json.NewDecoder(file).Decode(&configs)

		CachedKey = &configs.EventKey
	}
	return *CachedKey
}

func RegisterTeamColumns(key string) string {
	teamListInterface = nil
	writeTeamsToFile(key)

	numbers, err := os.ReadFile("TeamLists/$" + key)
	if err != nil {
		fmt.Println("File reading error", err)
		return ""
	}

	splitNumbers := strings.Split(string(numbers), "\n")
	name := splitNumbers[0]
	splitNumbers = splitNumbers[1:]

	maxColumn = len(splitNumbers) + 1
	for i, num := range splitNumbers {
		teamInt, _ := strconv.ParseInt(num, 10, 64)

		if teamInt != 0 {
			teamColumns[int(teamInt)] = fmt.Sprintf("A%v", (i + 2))
			teamListInterface = append(teamListInterface, []interface{}{teamInt})
		}
	}

	return name
}

func writeTeamsToFile(key string) {
	//You may need to replace the python with a different one
	runnable := exec.Command("python3.11", "getTeamList.py", key)

	out, err := runnable.Output()

	if err != nil {
		fmt.Println("err: " + err.Error())
	}

	print(string(out))
}

func GetTeamWriteRow(teamNum int) string {
	return teamColumns[teamNum]
}

func GetFullTeamRange() string {
	return fmt.Sprintf("A%v:%v", 2, maxColumn)
}

func GetTeamListAsInterface() [][]interface{} {
	return teamListInterface
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

func GetCurrentEvent() string {
	return "2024mnmi2"
}

type EventConfig struct {
	EventKey  string `json:"Key"`
	EventName string `json:"Name"`
}
