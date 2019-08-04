package cmd

import (
	"log"

	"github.com/lbryio/chainquery/apiactions"
	"github.com/lbryio/chainquery/config"
	"github.com/lbryio/chainquery/daemon"
	"github.com/lbryio/chainquery/db"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/twilio"
	"github.com/spf13/cobra"
)

func init() {
	serveCmd.AddCommand(daemonCmd)
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run only Daemon routines",
	Long:  `Run only Daemon routines, without the API Server. Check github.com/lbryio/chainquery#what-does-chainquery-consist-of`,
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
		daemon.DoYourThing()
	},
}
