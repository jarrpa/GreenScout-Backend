package lib

import (
	greenlogger "GreenScoutBackend/greenLogger"
	"GreenScoutBackend/userDB"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type TeamData struct {
	TeamNumber    uint64            `json:"Team"`
	Match         MatchInfo         `json:"Match"`
	Scouter       string            `json:"Scouter"`
	DriverStation DriverStationData `json:"Driver Station"`
	Cycles        []Cycle           `json:"Cycles"`
	Positions     SpeakerPositions  `json:"Speaker Positions"`
	Pickups       PickupLocations   `json:"Pickup Locations"`
	Auto          AutoData          `json:"Auto"`
	Climb         ClimbingData      `json:"Climbing"`
	Trap          TrapData          `json:"Trap"`
	Misc          MiscData          `json:"Misc"`
	Penalties     []string          `json:"Penalties"`
	Rescouting    bool              `json:"Rescouting"`
	Notes         string            `json:"Notes"`
}

type MatchInfo struct {
	Number   uint `json:"Number"`
	IsReplay bool `json:"isReplay"`
}

type DriverStationData struct {
	IsBlue bool `json:"Is Blue"`
	Number int  `json:"Number"`
}

type Cycle struct {
	Time    float64 `json:"Time"`
	Type    string  `json:"Type"`
	Success bool    `json:"Success"`
}

type AutoData struct {
	Can    bool `json:"Can"`
	Scores int  `json:"Scores"`
	Misses int  `json:"Misses"`
	Ejects int  `json:"Ejects"`
}

type ClimbingData struct {
	Succeeded bool    `json:"Succeeded"`
	Time      float64 `json:"Time"`
}

type TrapData struct {
	Attempts int `json:"Attempts"`
	Score    int `json:"Score"`
}

type SpeakerPositions struct {
	Sides  bool `json:"sides"`
	Middle bool `json:"Middle"`
}

type PickupLocations struct {
	Ground bool `json:"ground"`
	Source bool `json:"source"`
}

type MiscData struct {
	Parked    bool `json:"Parked"`
	DC        bool `json:"Lost Communication"`
	LostTrack bool `json:"User Lost Track"`
	Disabled  bool `json:"Disabled"`
}

// Bool is had errors
func Parse(file string, hasBeenWritten bool) (TeamData, bool) {

	var path string
	if hasBeenWritten {
		path = filepath.Join("InputtedJson", "Written")
	} else {
		path = filepath.Join("InputtedJson", "In")
	}

	// Open file
	jsonFile, fileErr := os.Open(filepath.Join(path, file))

	// Handle any error opening the file
	if fileErr != nil {
		greenlogger.LogErrorf(fileErr, "Error opening JSON file %v", filepath.Join(path, file))
		return TeamData{}, true
	}

	// defer file closing
	defer jsonFile.Close()

	var teamData TeamData

	dataAsByte, readErr := io.ReadAll(jsonFile)

	if readErr != nil {
		greenlogger.LogErrorf(readErr, "Error reading JSON file %v", filepath.Join(path, file))
		return TeamData{}, true
	}

	//Deocding
	err := json.Unmarshal(dataAsByte, &teamData)

	//Deal with unmarshalling errors
	if err != nil {
		greenlogger.LogErrorf(err, "Error unmarshalling JSON data %v", string(dataAsByte))
		return TeamData{}, true
	}

	userDB.ModifyUserScore(teamData.Scouter, userDB.Increase, 1)

	return teamData, false
}

type MatchInfoRequest struct {
	Match         int  `json:"Match"`
	IsBlue        bool `json:"isBlue"`
	DriverStation int  `json:"DriverStation"`
}

func GetNameFromWritten(match MatchInfoRequest) string {
	var names []string

	filePattern := fmt.Sprintf("%s_%v_%s", GetCurrentEvent(), match.Match, GetDSString(match.IsBlue, uint(match.DriverStation)))

	written, err := os.ReadDir(filepath.Join("InputtedJson", "Written"))

	if err != nil {
		greenlogger.LogErrorf(err, "Error searching %v", filepath.Join("InputtedJson", "Written"))
		return "Err in searching!"
	}

	for _, file := range written {

		splitByUnder := strings.Split(file.Name(), "_")

		if len(splitByUnder) > 3 && filePattern == strings.Join(splitByUnder[:3], "_") {

			// Open file
			jsonFile, fileErr := os.Open(filepath.Join("InputtedJson", "Written", file.Name()+".json"))

			// Handle any error opening the file
			if fileErr != nil {
				greenlogger.LogErrorf(fileErr, "Error opening JSON file %v", filepath.Join("InputtedJson", "Written", file.Name()+".json"))
			}

			// defer file closing
			defer jsonFile.Close()

			var teamData TeamData

			dataAsByte, readErr := io.ReadAll(jsonFile)

			if readErr != nil {
				greenlogger.LogErrorf(readErr, filepath.Join("InputtedJson", "Written", file.Name()+".json"))
			}

			//Deocding
			err := json.Unmarshal(dataAsByte, &teamData)

			//Deal with unmarshalling errors
			if err != nil {
				greenlogger.LogErrorf(err, "Error unmarshalling JSON data %v", string(dataAsByte))
			}

			if teamData.Scouter != "" {
				names = append(names, teamData.Scouter)

			}
		}
	}

	if len(names) == 0 {
		return "No scouters found!"
	}

	return strings.Join(names, ", ")
}

type PitScoutingData struct {
	TeamNumber    int    `json:"Team"`
	PitIdentifier string `json:"Pit"`
	Scouter       string `json:"Scouter"`

	Drivetrain string           `json:"Drivetrain"`
	Sides      SpeakerPositions `json:"Sides"`
	Distance   DistanceData     `json:"Distance"`

	AutoScores     int  `json:"Auto Scores"`
	MiddleControls int  `json:"Middle Notes"`
	NoteDetection  bool `json:"Detection"`

	Cycles           int    `json:"Cycles"`
	DriverExperience int    `json:"Experience"`
	BotType          string `json:"Bot Type"`

	EndgameBehavior string  `json:"Endgame Behavior"`
	ClimbTime       float64 `json:"Climb Time"`

	Notes string `json:"Notes"`
}

type DistanceData struct {
	Can      bool    `json:"Can"`
	Distance float64 `json:"Distance"`
}

type HumanPlayerData struct {
	Position      int `json:"Position"`
	StageAccuracy int `json:"Stage Accuracy"`
}

func ParsePitScout(file string) (PitScoutingData, bool) {

	path := filepath.Join("InputtedJson", "In")

	// Open file
	jsonFile, fileErr := os.Open(filepath.Join(path, file))

	// Handle any error opening the file
	if fileErr != nil {
		greenlogger.LogErrorf(fileErr, "Error opening JSON file %v", filepath.Join(path, file))
		return PitScoutingData{}, true
	}

	// defer file closing
	defer jsonFile.Close()

	var pitData PitScoutingData

	dataAsByte, readErr := io.ReadAll(jsonFile)

	if readErr != nil {
		greenlogger.LogErrorf(readErr, "Error reading JSON file %v", filepath.Join(path, file))
		return PitScoutingData{}, true
	}

	//Deocding
	err := json.Unmarshal(dataAsByte, &pitData)

	//Deal with unmarshalling errors
	if err != nil {
		greenlogger.LogErrorf(err, "Error unmarshalling JSON data %v", string(dataAsByte))
		return PitScoutingData{}, true
	}

	userDB.ModifyUserScore(pitData.Scouter, userDB.Increase, 1)

	return pitData, false
}
