package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type TeamData struct {
	TeamNumber       uint64            `json:"Team"`
	Match            MatchInfo         `json:"Match"`
	Scouter          string            `json:"Scouter"`
	DriverStation    DriverStationData `json:"Driver Station"`
	Cycles           []Cycle           `json:"Cycles"`
	Amp              bool              `json:"Amp"`
	Speaker          bool              `json:"Speaker"`
	Positions        SpeakerPositions  `json:"Speaker Positions"`
	DistanceShooting DistanceShotData  `json:"Distance Shooting"`
	Auto             AutoData          `json:"Auto"`
	Climb            ClimbingData      `json:"EndGame"`
	Trap             TrapData          `json:"Trap"`
	Misc             MiscData          `json:"Misc"`
	Penalties        []string          `json:"Penalties"`
	Notes            string            `json:"Notes"`
}

type MatchInfo struct {
	Number   uint `json:"Number"`
	IsReplay bool `json:"isReplay"`
}

type DriverStationData struct {
	IsBlue bool `json:"Is Blue"`
	Number int  `json:"Number"`
}

type DistanceShotData struct {
	Can    bool `json:"Can"`
	Misses int  `json:"Misses"`
	Scores int  `json:"Scores"`
}

type Cycle struct {
	Time float64 `json:"Time"`
	Type string  `json:"Type"`
}

type AutoData struct {
	Can       bool `json:"Can"`
	Succeeded bool `json:"Succeeded"`
	Scores    int  `json:"Scores"`
	Misses    int  `json:"Misses"`
	Ejects    int  `json:"Ejects"`
}

type ClimbingData struct {
	Can  bool    `json:"Can"`
	Time float64 `json:"Time"`
}

type TrapData struct {
	Attempts int `json:"Attempts"`
	Score    int `json:"Score"`
}

type SpeakerPositions struct {
	Sides  bool `json:"sides"`
	Middle bool `json:"Middle"`
}

type MiscData struct {
	Parked    bool `json:"Parked"`
	DC        bool `json:"Lost Communication"`
	LostTrack bool `json:"User Lost Track"`
}

func Parse(file string) TeamData {

	// Open file
	jsonFile, fileErr := os.Open("InputtedJson/In/" + file) // just didn't want to use interpolation here lul

	// Handle any error opening the file
	if fileErr != nil {
		fmt.Println(fileErr)
	}

	fmt.Println("Successfully Opened " + file)

	// defer file closing
	defer jsonFile.Close()

	var teamData TeamData

	dataAsByte, readErr := io.ReadAll(jsonFile)

	if readErr != nil {
		fmt.Println(readErr)
	}

	//Decding
	err := json.Unmarshal(dataAsByte, &teamData)

	//Deal with unmarshalling errors
	if err != nil {
		fmt.Println(err)
	}

	return teamData
}
