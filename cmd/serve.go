package cmd

import (
	"log"

	"github.com/lbryio/chainquery/apiactions"
	"github.com/lbryio/chainquery/config"
	"github.com/lbryio/chainquery/daemon"
	"github.com/lbryio/chainquery/db"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/swagger/apiserver"
	"github.com/lbryio/chainquery/twilio"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run Daemon routines and the API Server",
	Long:  `Run Daemon routines and the API Server, check github.com/lbryio/chainquery#what-does-chainquery-consist-of`,
	Args:  cobra.OnlyValidArgs,
	Run: func(cmd *cobra.Command, args []string) {
		config.InitSlack()
		twilio.InitTwilio()
		apiactions.AutoUpdateCommand = config.GetAutoUpdateCommand()
		//Main Chainquery DB connection
		dbInstance, err := db.Init(config.GetMySQLDSN(), config.GetDebugQueryMode())
		if err != nil {
			log.Panic(err)
		}
		defer db.CloseDB(dbInstance)

		lbrycrdClient := lbrycrd.Init()
		defer lbrycrdClient.Shutdown()

		go swagger.InitApiServer(config.GetAPIHostAndPort())
		daemon.DoYourThing()
	},
}
