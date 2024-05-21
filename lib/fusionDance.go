package lib

import (
	"fmt"
	"math"

	"github.com/montanaflynn/stats"
)

type MultiMatch struct {
	TeamNumber       uint64    `json:"Team"`
	Match            MatchInfo `json:"Match"`
	Scouters         string
	DriverStation    DriverStationData `json:"Driver Station"`
	CycleData        CompositeCycleData
	SpeakerPositions SpeakerPositions
	Pickups          PickupLocations
	Auto             AutoData
	Climb            ClimbingData
	Parked           bool
	TrapScore        int

	Notes []string
}

type CompositeCycleData struct {
	NumCycles    int
	AvgCycleTime float64
	AllCycles    []Cycle

	HadMismatches bool
}

type TypeData struct {
	Tendency      float64
	Accuracy      float64
	HadMismatches bool
}

func CompileMultiMatch(entries ...TeamData) MultiMatch {
	var finalData MultiMatch

	teamNum, _ := compositeTeamNum(entries)

	finalData.TeamNumber = uint64(teamNum)

	finalData.Match = entries[0].Match

	finalData.Scouters = compositeScouters(entries)

	finalData.DriverStation = entries[0].DriverStation

	finalData.CycleData = compileCycles(entries)

	finalData.SpeakerPositions = compileSpeakerPositions(entries)

	finalData.Pickups = compilePickupPositions(entries)

	finalData.Auto = compileAutoData(entries)

	finalData.Climb = compileClimbData(entries)

	finalData.Parked = compileParked(entries)

	finalData.TrapScore = compileTrapScore(entries)

	finalData.Notes = compileNotes(entries, nil)

	return finalData
}

func compositeTeamNum(entries []TeamData) (int, bool) {
	initial := entries[0].TeamNumber

	for i := 1; i < len(entries); i++ {
		if initial != entries[i].TeamNumber {
			return int(initial), true
		}
	}

	return int(initial), false
}

func compositeScouters(entries []TeamData) string {
	var finalScouter string
	for _, entry := range entries {
		finalScouter += fmt.Sprintf(", %s", entry.Scouter)
	}

	return finalScouter
}

func compileCycles(entries []TeamData) CompositeCycleData {
	var finalCycles CompositeCycleData
	var allNumCycles []int
	for _, entry := range entries {
		allNumCycles = append(allNumCycles, GetNumCycles(entry.Cycles))
	}

	for _, cycleNum := range allNumCycles {
		if cycleNum != allNumCycles[0] {
			finalCycles.HadMismatches = true
		}
	}

	cycleCompositeTime, hadMismatches := avgCycleTimes(entries)

	finalCycles.AvgCycleTime = cycleCompositeTime

	if hadMismatches {
		finalCycles.HadMismatches = true
	}

	var massiveBlockOfCycles []Cycle
	for _, entry := range entries {
		massiveBlockOfCycles = append(massiveBlockOfCycles, entry.Cycles...)
	}

	finalCycles.AllCycles = massiveBlockOfCycles

	return finalCycles
}

func avgCycleTimes(entries []TeamData) (float64, bool) {
	var sum float64
	var count int = 0

	var allCycles [][]Cycle

	for _, entry := range entries {
		allCycles = append(allCycles, entry.Cycles)
		entryAvg := GetAvgCycleTimeExclusive(entry.Cycles)
		if entryAvg != 0 {
			sum += entryAvg
			count++
		}
	}

	finalAvg := sum / float64(count)

	if math.IsNaN(finalAvg) {
		finalAvg = 0
	}
	return finalAvg, !CompareCycles(allCycles)
}

func compileSpeakerPositions(entries []TeamData) SpeakerPositions {
	var sides bool = false
	var middle bool = false

	for _, entry := range entries {
		if entry.Positions.Sides {
			sides = true
		}

		if entry.Positions.Middle {
			middle = true
		}
	}

	return SpeakerPositions{
		Sides:  sides,
		Middle: middle,
	}
}

func compilePickupPositions(entries []TeamData) PickupLocations {
	var ground bool = false
	var source bool = false

	for _, entry := range entries {
		if entry.Pickups.Ground {
			ground = true
		}

		if entry.Pickups.Source {
			source = true
		}
	}

	return PickupLocations{
		Ground: ground,
		Source: source,
	}
}

func compileAutoData(entries []TeamData) AutoData {
	var can bool = false
	var allScores []float64
	var allMisses []float64
	var allEjects []float64

	for _, entry := range entries {
		if entry.Auto.Can {
			can = true
		}

		allScores = append(allScores, float64(entry.Auto.Scores))
		allMisses = append(allMisses, float64(entry.Auto.Misses))
		allEjects = append(allEjects, float64(entry.Auto.Ejects))
	}

	scoresAvgd, _ := stats.Mean(allScores)
	missesAvgd, _ := stats.Mean(allMisses)
	ejectsAvgd, _ := stats.Mean(allEjects)

	return AutoData{
		Can:    can,
		Scores: int(scoresAvgd),
		Misses: int(missesAvgd),
		Ejects: int(ejectsAvgd),
	}
}

func compileClimbData(entries []TeamData) ClimbingData {
	var success bool = false
	var times []float64

	for _, entry := range entries {
		if entry.Climb.Succeeded {
			success = true
		}

		if entry.Climb.Time > 0 {
			times = append(times, entry.Climb.Time)
		}
	}

	timeAvgd, _ := stats.Mean(times)
	return ClimbingData{
		Succeeded: success,
		Time:      timeAvgd,
	}
}

func compileParked(entries []TeamData) bool {
	for _, entry := range entries {
		if entry.Misc.Parked {
			return true
		}
	}
	return false
}

func compileTrapScore(entries []TeamData) int {
	var trapScores []float64
	for _, entry := range entries {
		trapScores = append(trapScores, float64(entry.Trap.Score))
	}

	trapAvgd, _ := stats.Mean(trapScores)

	return int(math.Round(trapAvgd))
}

func compileNotes(entries []TeamData, mismatches []string) []string {
	var finalNotes []string
	for _, entry := range entries {
		finalNotes = append(finalNotes, entry.Notes)
		finalNotes = append(finalNotes, mismatches...)
	}
	return finalNotes
}
