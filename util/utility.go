package util

import (
	"log"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

func Plus(decimal string, value float64) string {
	deciValue, _ := strconv.ParseFloat(decimal, 64)
	deciValue = deciValue + value
	deciString := strconv.FormatFloat(deciValue, 'f', -1, 64)

	return deciString
}

func TimeTrack(start time.Time, name string, profile string) {
	if profile == "daemonprofile" && viper.GetBool("daemonprofile") {
		elapsed := time.Since(start)
		log.Printf("%s %s took %s", name, profile, elapsed)
	}
	if profile == "lbrycrdprofile" && viper.GetBool("lbrycrdprofile") {
		elapsed := time.Since(start)
		log.Printf("%s %s took %s", name, profile, elapsed)
	}
	if profile == "mysqlprofile" && viper.GetBool("mysqlprofile") {
		elapsed := time.Since(start)
		log.Printf("%s %s took %s", name, profile, elapsed)
	}
	if profile == "always" {
		elapsed := time.Since(start)
		log.Printf("%s %s took %s", name, profile, elapsed)
	}

}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
