package util

import (
	"log"
	"strconv"
	"time"
)

func Plus(decimal string, value float64) string {
	deciValue, _ := strconv.ParseFloat(decimal, 64)
	deciValue = deciValue + value
	deciString := strconv.FormatFloat(deciValue, 'f', -1, 64)

	return deciString
}

func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}
