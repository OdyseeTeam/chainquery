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

// rev reverses a byte slice. useful for switching endian-ness
func ReverseBytes(b []byte) []byte {
	r := make([]byte, len(b))
	for left, right := 0, len(b)-1; left < right; left, right = left+1, right-1 {
		r[left], r[right] = b[right], b[left]
	}
	return r
}
