package lib

import (
	greenlogger "GreenScoutBackend/greenLogger"
	"math"

	"github.com/montanaflynn/stats"
)

const kAllowableSeconds = 1.0

// Returns if they were close enough
func CompareCycles(data [][]Cycle) bool {

	var averages []float64
	for _, entry := range data {
		averages = append(averages, entry[len(entry)-1].Time/float64(len(entry)))
	}

	return isNearSeconds(averages, kAllowableSeconds)
}

func isNearSeconds(averages []float64, allowableErr float64) bool {
	//If any err, returns false to draw attention to itself

	max, maxErr := stats.Max(averages)
	if maxErr != nil {
		greenlogger.LogErrorf(maxErr, "Error finding maximum of %v", averages)
		return false
	}
	min, minErr := stats.Min(averages)
	if minErr != nil {
		greenlogger.LogErrorf(minErr, "Error finding minimum of %v", averages)
		return false
	}

	return math.Abs(max-min) <= allowableErr
}
