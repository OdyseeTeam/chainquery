package cmd

import (
	"log"

	"github.com/lbryio/chainquery/config"
	"github.com/lbryio/chainquery/db"
	"github.com/lbryio/chainquery/lbrycrd"
	swagger "github.com/lbryio/chainquery/swagger/apiserver"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(apiCmd)
}

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Start only the api server",
	Long:  `This runs the API Server for chainquery only. The daemon does not run, however, the db is still required and all APIs are available.`,
	Run: func(cmd *cobra.Command, args []string) {
		config.InitSlack()
		//Main Chainquery DB connection
		dbInstance, err := db.Init(config.GetMySQLDSN(), config.GetDebugQueryMode())
		if err != nil {
			log.Panic(err)
		}
		defer db.CloseDB(dbInstance)

		lbrycrd.Init()
		swagger.InitApiServer(config.GetAPIHostAndPort())
	},
}
