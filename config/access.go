package config

import (
	"time"

	"github.com/lbryio/lbry.go/errors"

	"github.com/spf13/viper"
)

// GetMySQLDSN gets the MySql DSN from viper configuration
func GetMySQLDSN() string {
	return viper.GetString(mysqldsn)
}

// GetLBRYcrdURL gets the LBRYcrd URL from viper configuration
func GetLBRYcrdURL() string {
	if viper.IsSet(lbrycrdurl) {
		return viper.GetString(lbrycrdurl)
	}
	url, err := getLbrycrdURLFromConfFile()
	if err != nil {
		err = errors.Prefix("LBRYcrd conf file error: ", err)
		panic(err)
	}
	return url
}

// GetDefaultClientTimeout gets the default client timeout.
func GetDefaultClientTimeout() time.Duration {
	return viper.GetDuration(defaultclienttimeout)
}

// GetDaemonMode gets the daemon mode from the viper configuration. See default toml file for different modes.
func GetDaemonMode() int {
	return viper.GetInt(daemonmode)
}

// GetProcessingDelay gets the processing delay from the viper configuration.
func GetProcessingDelay() time.Duration {
	return viper.GetDuration(processingdelay) * time.Millisecond
}

// GetDaemonDelay gets the deamon delay from the viper configuration
func GetDaemonDelay() time.Duration {
	return viper.GetDuration(daemondelay) * time.Second
}
