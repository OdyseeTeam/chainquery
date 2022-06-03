package config

import (
	"time"

	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// GetMySQLDSN gets the MySql DSN from viper configuration
func GetMySQLDSN() string {
	return viper.GetString(mysqldsn)
}

//GetAPIMySQLDSN gets the API MySql DSN from viper. This intended to be another account with limited privileges
//for the apis to prevent potential abuse. It should have read only privileges at a minimum.
func GetAPIMySQLDSN() string {
	return viper.GetString(apimysqldsn)
}

// GetLBRYcrdURL gets the LBRYcrd URL from viper configuration
func GetLBRYcrdURL() string {
	if viper.IsSet(lbrycrdurl) {
		return viper.GetString(lbrycrdurl)
	}
	url, err := getLbrycrdURLFromConfFile()
	if err != nil {
		err = errors.Prefix("LBRYcrd conf file error: ", err)
		logrus.Panic(err)
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

// GetAPIHostAndPort gets the host and port string the api server should bind and listen too.
func GetAPIHostAndPort() string {
	return viper.GetString(apihostport)
}

// GetDebugMode returns true/false if the app is in debug mode.
func GetDebugMode() bool {
	return viper.GetBool(debugmode)
}

// GetDebugQueryMode turns on SQLBoiler query output
func GetDebugQueryMode() bool {
	return viper.GetBool(debugquerymode)
}

// GetAutoUpdateCommand returns the command that should be executed to trigger a self update of the software.
func GetAutoUpdateCommand() string {
	return viper.GetString(autoupdatecommand)
}
