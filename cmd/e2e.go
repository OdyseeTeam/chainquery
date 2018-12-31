package cmd

import (
	"log"

	"github.com/lbryio/chainquery/config"
	"github.com/lbryio/chainquery/db"
	"github.com/lbryio/chainquery/lbrycrd"

	"github.com/lbryio/chainquery/e2e"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(e2eCmd)
}

var e2eCmd = &cobra.Command{
	Use:   "e2e",
	Short: "Launch end to end tests",
	Long:  `Launch end to end tests`,
	Run: func(cmd *cobra.Command, args []string) {
		//Main Chainquery DB connection
		dbInstance, err := db.Init(config.GetMySQLDSN(), config.GetDebugQueryMode())
		if err != nil {
			log.Panic(err)
		}
		defer db.CloseDB(dbInstance)

		lbrycrdClient := lbrycrd.Init()
		defer lbrycrdClient.Shutdown()
		e2e.StartE2ETesting()
	},
}
