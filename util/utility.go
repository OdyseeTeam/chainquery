package util

import (
	"database/sql"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// TimeTrack is a function that tracks the time spent and outputs specific timing information. This is important for
// chainquery profiling and is used throughout. It can be reused by just passing `always` as the profile. The basic
// usage is `defer util.TimeTrack(time.Now(),"<useful identifier>","<profile>")`. This should be placed at the top of
// the function where time is to be tracked, or at any point where you want to start tracking time.
func TimeTrack(start time.Time, name string, profile string) {
	if profile == "daemonprofile" && viper.GetBool("daemonprofile") {
		elapsed := time.Since(start)
		logrus.Infof("%s %s took %s", name, profile, elapsed)
	}
	if profile == "lbrycrdprofile" && viper.GetBool("lbrycrdprofile") {
		elapsed := time.Since(start)
		logrus.Infof("%s %s took %s", name, profile, elapsed)
	}
	if profile == "mysqlprofile" && viper.GetBool("mysqlprofile") {
		elapsed := time.Since(start)
		logrus.Infof("%s %s took %s", name, profile, elapsed)
	}
	if profile == "always" {
		elapsed := time.Since(start)
		logrus.Infof("%s took %s", name, elapsed)
	}

}

// Min is a helpful function to take the min between two integers.
func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

//CloseRows Closes SQL Rows for custom SQL queries.
func CloseRows(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		logrus.Error("Closing rows error: ", err)
	}
}
