package config

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/lbryio/chainquery/sockety"

	"github.com/lbryio/chainquery/notifications"

	"github.com/lbryio/chainquery/apiactions"
	"github.com/lbryio/chainquery/auth"
	"github.com/lbryio/chainquery/daemon"
	"github.com/lbryio/chainquery/daemon/jobs"
	"github.com/lbryio/chainquery/daemon/processing"
	"github.com/lbryio/chainquery/global"
	"github.com/lbryio/chainquery/lbrycrd"
	server "github.com/lbryio/chainquery/swagger/apiserver/go"
	"github.com/lbryio/chainquery/twilio"

	"github.com/lbryio/lbry.go/extras/errors"

	"github.com/fsnotify/fsnotify"
	"github.com/go-ini/ini"
	"github.com/johntdyer/slackrus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/boil"
)

const ( // config setting keys
	debugmode                 = "debugmode"
	debugquerymode            = "debugquerymode"
	mysqldsn                  = "mysqldsn"
	apimysqldsn               = "apimysqldsn"
	lbrycrdurl                = "lbrycrdurl"
	profilemode               = "profilemode"
	daemonmode                = "daemonmode"
	processingdelay           = "processingdelay"
	daemondelay               = "daemondelay"
	defaultclienttimeout      = "defaultclienttimeout"
	daemonprofile             = "daemonprofile"
	lbrycrdprofile            = "lbrycrdprofile"
	mysqlprofile              = "mysqlprofile"
	apihostport               = "apihostport"
	slackhookurl              = "slackhookurl"
	slackchannel              = "slackchannel"
	slackloglevel             = "slackloglevel"
	autoupdatecommand         = "autoupdatecommand"
	twiliosid                 = "twiliosid"
	twilioauthtoken           = "twilioauthtoken"
	smsrecipients             = "smsrecipients"
	smsfromphonenumber        = "smsfromphonenumber"
	apikeys                   = "apikeys"
	maxfailures               = "maxfailures"
	blockchainname            = "blockchainname"
	chainsyncrunduration      = "chainsyncrunduration"
	chainsyncdelay            = "chainsyncdelay"
	maxsqlapitimeout          = "maxsqlapitimeout"
	maxparalleltxprocessing   = "maxparalleltxprocessing"
	maxparallelvinprocessing  = "maxparallelvinprocessing"
	maxparallelvoutprocessing = "maxparallelvoutprocessing"
	promuser                  = "promuser"
	prompass                  = "prompass"
	socketytoken              = "socketytoken"
	socketyurl                = "socketyurl"
)

const (
	//Chainquery Flags
	configpathflag  = "configpath"
	reindexflag     = "reindex"
	reindexwipeflag = "reindexwipe"
	debugmodeflag   = "debug"
	tracemodeflag   = "trace"
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
	pflag.BoolP(debugmodeflag, "d", false, "turns on debug mode for the application command.")
	pflag.BoolP(tracemodeflag, "t", false, "turns on trace mode for the application command, very verbose logging.")
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
	err := viper.ReadInConfig()                          // Find and read the config file
	if err != nil {                                      // Handle errors reading the config file
		logrus.Warning("Error reading config file...defaults will be used: ", err)
	}
	twilio.RecipientList = viper.GetStringSlice(smsrecipients)
	notifications.ClearSubscribers()
	subscriptions := viper.GetStringMap("subscription")
	err = applySubscribers(subscriptions)
	if err != nil {
		logrus.Error("could not apply subsribers: ", err)
	}
	twilio.FromNumber = viper.GetString(smsfromphonenumber)
	twilio.TwilioAuthToken = viper.GetString(twilioauthtoken)
	twilio.TwilioSID = viper.GetString(twiliosid)
}

func initDefaults() {
	//Setting viper defaults in the event there are not settings set in the config file.
	viper.SetDefault(debugmode, false)
	viper.SetDefault(debugquerymode, false)
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
	viper.SetDefault("codeprofile", false)
	viper.SetDefault(apihostport, "0.0.0.0:6300")
	viper.SetDefault(slackloglevel, 0)
	viper.SetDefault(smsrecipients, []string{})
	viper.SetDefault(smsfromphonenumber, "")
	viper.SetDefault(maxfailures, 1000)
	viper.SetDefault(blockchainname, "lbrycrd_main")
	viper.SetDefault(chainsyncrunduration, 60)
	viper.SetDefault(chainsyncdelay, 100)
	viper.SetDefault(maxsqlapitimeout, 5)
	viper.SetDefault(maxparalleltxprocessing, runtime.NumCPU())
	viper.SetDefault(maxparallelvinprocessing, runtime.NumCPU())
	viper.SetDefault(maxparallelvoutprocessing, runtime.NumCPU())
}

func processConfiguration() {
	// Things that listen live for setting changes that need to be applied. Settings that are retrieved do not need
	// to be set here.
	isdebug := viper.GetBool(debugmode)
	if isdebug {
		logrus.Info("SETTINGS: debug mode turned on")
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	if boil.DebugMode != viper.GetBool(debugquerymode) {
		logrus.Info("SETTINGS: changing query debug mode to ", viper.GetBool(debugquerymode))
		boil.DebugMode = viper.GetBool(debugquerymode)
	}

	settings := global.DaemonSettings{
		DaemonMode:      GetDaemonMode(),
		ProcessingDelay: GetProcessingDelay(),
		DaemonDelay:     GetDaemonDelay(),
		IsReIndex:       viper.GetBool(reindexflag)}

	daemon.ApplySettings(settings)
	lbrycrd.LBRYcrdURL = GetLBRYcrdURL()
	auth.APIKeys = viper.GetStringSlice(apikeys)
	processing.MaxFailures = viper.GetInt(maxfailures)
	processing.MaxParallelTxProcessing = viper.GetInt(maxparalleltxprocessing)
	processing.MaxParallelVinProcessing = viper.GetInt(maxparallelvinprocessing)
	processing.MaxParallelVoutProcessing = viper.GetInt(maxparallelvoutprocessing)
	global.BlockChainName = viper.GetString(blockchainname)
	jobs.ChainSyncDelay = viper.GetInt(chainsyncdelay)
	jobs.ChainSyncRunDuration = viper.GetInt(chainsyncrunduration)
	apiactions.MaxSQLAPITimeout = viper.GetInt(maxsqlapitimeout)
	server.PromUser = viper.GetString(promuser)
	server.PromPassword = viper.GetString(prompass)
	sockety.Token = viper.GetString(socketytoken)
	sockety.URL = viper.GetString(socketyurl)

	//Flags last so they override everything before, even config
	if viper.IsSet(debugmodeflag) {
		if viper.GetBool(debugmodeflag) {
			logrus.SetLevel(logrus.DebugLevel)
		}
	}
	if viper.IsSet(tracemodeflag) {
		if viper.GetBool(tracemodeflag) {
			logrus.SetLevel(logrus.TraceLevel)
		}
	}

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

func applySubscribers(subs map[string]interface{}) error {
	for subType, p := range subs {
		typeSubsInt, ok := p.([]interface{})
		if ok {
			for _, typeSub := range typeSubsInt {
				params, ok := typeSub.(map[string]interface{})
				if ok {
					url, ok := params["url"].(string)
					if ok {
						delete(params, "url")
						notifications.AddSubscriber(url, subType, params)
					} else {
						return errors.Err("url is required")
					}
				} else {
					return errors.Err("could not find params map for the subscription type instance")
				}
			}
		} else {
			return errors.Err("could not find sub type array under subscription")
		}
	}
	return nil
}
