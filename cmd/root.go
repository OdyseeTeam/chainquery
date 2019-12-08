package cmd

import (
	"fmt"
	"os"

	"net/http"

	"github.com/lbryio/chainquery/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const ( // config setting keys
	configpathflag = "configpath"
	debugmodeflag  = "debug"
	tracemodeflag  = "trace"
)

func init() {
	cobra.OnInitialize(config.InitializeConfiguration)
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	http.DefaultClient.Timeout = config.GetDefaultClientTimeout()
	rootCmd.PersistentFlags().String(configpathflag, "", "Specify non-default location of the configuration of chainquery. The precedence is $HOME, working directory, and lastly the branch path to the default configuration 'path/to/chainquery/config/default/'")
	rootCmd.PersistentFlags().BoolP(debugmodeflag, "d", false, "turns on debug mode for the application command.")
	rootCmd.PersistentFlags().BoolP(tracemodeflag, "t", false, "turns on trace mode for the application command, very verbose logging.")
	err := viper.BindPFlags(rootCmd.PersistentFlags())
	if err != nil {
		log.Panic(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "chainquery",
	Short: "Chainquery parses and syncs the LBRY blockchain data into structured SQL",
	Long: `Chainquery uses a MySQL server instance and a LBRYcrd instance to parse the blockchain
			into data that can be queried quickly and easily`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

	},
}

// Execute executes the root command and is the entry point of the application from main.go
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
