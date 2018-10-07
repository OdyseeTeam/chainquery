package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/lbryio/chainquery/daemon"
	"github.com/lbryio/chainquery/global"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/twilio"
	"github.com/lbryio/lbry.go/errors"

	"github.com/fsnotify/fsnotify"
	"github.com/go-ini/ini"
	"github.com/johntdyer/slackrus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const ( // config setting keys
	debugmode            = "debugmode"
	mysqldsn             = "mysqldsn"
	apimysqldsn          = "apimysqldsn"
	lbrycrdurl           = "lbrycrdurl"
	profilemode          = "profilemode"
	daemonmode           = "daemonmode"
	processingdelay      = "processingdelay"
	daemondelay          = "daemondelay"
	defaultclienttimeout = "defaultclienttimeout"
	daemonprofile        = "daemonprofile"
	lbrycrdprofile       = "lbrycrdprofile"
	mysqlprofile         = "mysqlprofile"
	apihostport          = "apihostport"
	slackhookurl         = "slackhookurl"
	slackchannel         = "slackchannel"
	slackloglevel        = "slackloglevel"
	autoupdatecommand    = "autoupdatecommand"
	twiliosid            = "twiliosid"
	twilioauthtoken      = "twilioauthtoken"
	smsrecipients        = "smsrecipients"
	smsfromphonenumber   = "smsfromphonenumber"
)

const (
	//Chainquery Flags
	configpathflag  = "configpath"
	reindexflag     = "reindex"
	reindexwipeflag = "reindexwipe"
)

// InitializeConfiguration is the main entry point from outside the package. This initializes the configuration and watcher.
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

// InitSlack initializes the slack connection and posts info level or greater to the set channel.
func InitSlack() {
	slackURL := viper.GetString(slackhookurl)
	slackChannel := viper.GetString(slackchannel)
	slackLogLevel := viper.GetInt(slackloglevel)
	if slackURL != "" && slackChannel != "" {
		logrus.AddHook(&slackrus.SlackrusHook{
			HookURL:        slackURL,
			AcceptedLevels: slackrus.LevelThreshold(logrus.Level(slackLogLevel)),
			Channel:        slackChannel,
			IconEmoji:      ":golang:",
			Username:       "Chainquery",
		})
	}
}

func initFlags() {
	// using standard library "flag" package
	pflag.BoolP(reindexflag, "r", false, "Rebuilds the database from the 1st block. Does not wipe the database")
	pflag.BoolP(reindexwipeflag, "w", false, "Drops all tables and rebuilds the database from the 1st block.")
	pflag.StringP(configpathflag, "c", "", "Specify non-default location of the configuration of chainquery. The precedence is $HOME, working directory, and lastly the branch path to the default configuration 'path/to/chainquery/config/default/'")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		logrus.Panic(err)
	}

}

func readConfig() {
	viper.SetConfigName("chainqueryconfig")              // name of config file (without extension)
	viper.AddConfigPath(viper.GetString(configpathflag)) // 1 - commandline config path
	viper.AddConfigPath("$HOME/")                        // 2 - check $HOME
	viper.AddConfigPath(".")                             // 3 - optionally look for config in the working directory
	viper.AddConfigPath("./config/default/")             // 4 - use default that comes with the branch

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		logrus.Warning("Error reading config file...defaults will be used: ", err)
	}
	twilio.RecipientList = viper.GetStringSlice(smsrecipients)
	twilio.FromNumber = viper.GetString(smsfromphonenumber)
	twilio.TwilioAuthToken = viper.GetString(twilioauthtoken)
	twilio.TwilioSID = viper.GetString(twiliosid)
}

func initDefaults() {
	//Setting viper defaults in the event there are not settings set in the config file.
	viper.SetDefault(debugmode, false)
	viper.SetDefault(mysqldsn, "lbry:lbry@tcp(localhost:3306)/chainquery")
	viper.SetDefault(apimysqldsn, "lbry:lbry@tcp(localhost:3306)/chainquery")
	viper.SetDefault(lbrycrdurl, "rpc://lbry:lbry@localhost:9245")
	viper.SetDefault(profilemode, false)
	viper.SetDefault(daemonmode, 0) //BEASTMODE
	viper.SetDefault(defaultclienttimeout, 20*time.Second)
	viper.SetDefault(daemondelay, 1)       //Seconds
	viper.SetDefault(processingdelay, 100) //Milliseconds
	viper.SetDefault(daemonprofile, false)
	viper.SetDefault(lbrycrdprofile, false)
	viper.SetDefault(mysqlprofile, false)
	viper.SetDefault(apihostport, "0.0.0.0:6300")
	viper.SetDefault(slackloglevel, 0)
	viper.SetDefault(smsrecipients, []string{})
	viper.SetDefault(smsfromphonenumber, "")
}

func processConfiguration() {
	// Things that listen live for setting changes that need to be applied. Settings that are retrieved do not need
	// to be set here.
	isdebug := viper.GetBool(debugmode)
	if isdebug {
		logrus.Info("Setting DebugMode=true")
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	settings := global.DaemonSettings{
		DaemonMode:      GetDaemonMode(),
		ProcessingDelay: GetProcessingDelay(),
		DaemonDelay:     GetDaemonDelay(),
		IsReIndex:       viper.GetBool(reindexflag)}

	daemon.ApplySettings(settings)
	lbrycrd.LBRYcrdURL = GetLBRYcrdURL()

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
