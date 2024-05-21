package lib

import (
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

	max, _ := stats.Max(averages)
	min, _ := stats.Min(averages)

	return math.Abs(max-min) <= allowableErr
}
