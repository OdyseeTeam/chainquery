package util

import "strconv"

func Plus(decimal string, value float64) string {
	deciValue, _ := strconv.ParseFloat(decimal, 64)
	deciValue = deciValue + value
	deciString := strconv.FormatFloat(deciValue, 'f', -1, 64)

	return deciString
}
