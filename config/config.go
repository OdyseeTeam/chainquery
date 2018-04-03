package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/lbryio/chainquery/daemon"
	"github.com/lbryio/errors.go"

	"github.com/fsnotify/fsnotify"
	"github.com/go-ini/ini"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const ( // config setting keys
	DEBUGMODE            = "debugmode"
	MYSQLDSN             = "mysqldsn"
	LBRYCRDURL           = "lbrycrdurl"
	PROFILEMODE          = "profilemode"
	DAEMONMODE           = "daemonmode"
	PROCESSINGDELAY      = "processingdelay"
	DAEMONDELAY          = "daemondelay"
	DEFAULTCLIENTTIMEOUT = "defaultclienttimeout"
)

const ( //Flags
	CONFIGPATHFLAG = "configpath"
)

func InitializeConfiguration() {
	initDefaults()
	initFlags()
	readConfig()
	processConfiguration()
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
		readConfig()
		processConfiguration()
	})

}

func initFlags() {
	// using standard library "flag" package
	flag.Int(CONFIGPATHFLAG, 1234, "Specify non-default location of the configuration of chainquery.")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)
}

func readConfig() {
	viper.SetConfigName("chainqueryconfig")  // name of config file (without extension)
	viper.AddConfigPath("/etc/chainquery/")  // check for it in etc folder
	viper.AddConfigPath("$HOME/")            // check $HOME
	viper.AddConfigPath(".")                 // optionally look for config in the working directory
	viper.AddConfigPath("./config/default/") // use default that comes with the branch
	viper.AddConfigPath(viper.GetString(CONFIGPATHFLAG))

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		logrus.Warning("Error reading config file...defaults will be used")
	}
}

func initDefaults() {
	viper.SetDefault(DEBUGMODE, false)
	viper.SetDefault(MYSQLDSN, "lbry:lbry@tcp(localhost:3306)/lbrycrd")
	viper.SetDefault(LBRYCRDURL, "rpc://lbry:lbry@localhost:9245")
	viper.SetDefault(PROFILEMODE, false)
	viper.SetDefault(DAEMONMODE, daemon.BEASTMODE)
	viper.SetDefault(DEFAULTCLIENTTIMEOUT, 20*time.Second)
	viper.SetDefault(DAEMONDELAY, 1)       //Seconds
	viper.SetDefault(PROCESSINGDELAY, 100) //Milliseconds

}

func processConfiguration() {
	isdebug := viper.GetBool(DEBUGMODE)
	if isdebug {
		logrus.Info("Setting DebugMode=true")
		logrus.SetLevel(logrus.DebugLevel)
	}
	daemon.ProcessingMode = GetDaemonMode()
	logrus.Info("Daemon mode = ", GetDaemonMode())
	daemon.ApplySettings(GetProcessingDelay(), GetDaemonDelay())
}

func getLbrycrdURLFromConfFile() (string, error) {
	if os.Getenv("HOME") == "" {
		return "", errors.Err("$HOME env var not set")
	}

	defaultConfFile := os.Getenv("HOME") + "/.lbrycrd/lbrycrd.conf"
	if _, err := os.Stat(defaultConfFile); os.IsNotExist(err) {
		return "", errors.Err("lbrycrd conf file not found")
	}

	cfg, err := ini.Load(defaultConfFile)
	if err != nil {
		return "", errors.Err(err)
	}

	section, err := cfg.GetSection("")
	if err != nil {
		return "", errors.Err(err)
	}

	username := section.Key("rpcuser").String()
	password := section.Key("rpcpassword").String()
	host := section.Key("rpchost").String()
	if host == "" {
		host = "localhost"
	}
	port := section.Key("rpcport").String()
	if port == "" {
		port = ":9245"
	} else {
		port = ":" + port
	}

	userpass := ""
	if username != "" || password != "" {
		userpass = username + ":" + password + "@"
	}

	return "rpc://" + userpass + host + port, nil
}
