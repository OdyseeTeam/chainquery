package config

import (
	"time"

	"github.com/lbryio/lbry.go/errors"

	"github.com/spf13/viper"
)

func GetMySQLDSN() string {
	return viper.GetString(MYSQLDSN)
}

func GetLBRYcrdURL() string {
	if viper.IsSet(LBRYCRDURL) {
		return viper.GetString(LBRYCRDURL)
	}
	url, err := getLbrycrdURLFromConfFile()
	if err != nil {
		err = errors.Prefix("LBRYcrd config file error: ", err)
		panic(err)
	}
	return url
}

func GetDefaultClientTimeout() time.Duration {
	return viper.GetDuration(DEFAULTCLIENTTIMEOUT)
}

func GetDaemonMode() int {
	return viper.GetInt(DAEMONMODE)
}

func GetProcessingDelay() time.Duration {
	return viper.GetDuration(PROCESSINGDELAY) * time.Millisecond
}

func GetDaemonDelay() time.Duration {
	return viper.GetDuration(DAEMONDELAY) * time.Second
}
