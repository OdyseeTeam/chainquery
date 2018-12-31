package cmd

import (
	"log"

	"github.com/lbryio/chainquery/config"
	"github.com/lbryio/chainquery/db"
	"github.com/spf13/cobra"
)

func init() {
	serveCmd.AddCommand(dbCmd)
}

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Sets up the database",
	Long:  `This will initialize the specified database schema`,
	Run: func(cmd *cobra.Command, args []string) {
		//Main Chainquery DB connection
		dbInstance, err := db.Init(config.GetMySQLDSN(), config.GetDebugQueryMode())
		if err != nil {
			log.Panic(err)
		}
		db.CloseDB(dbInstance)
	},
}
