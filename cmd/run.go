package cmd

import (
	"log"
	"strings"

	"github.com/pkg/profile"
	"github.com/spf13/viper"

	"github.com/lbryio/chainquery/config"
	"github.com/lbryio/chainquery/daemon/jobs"
	"github.com/lbryio/chainquery/db"
	"github.com/lbryio/chainquery/lbrycrd"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var jobsMap = map[string]func(){
	"claimtrie":        jobs.ClaimTrieSync,
	"certificate":      jobs.CertificateSync,
	"mempool":          jobs.MempoolSync,
	"transactionvalue": jobs.TransactionValueSync,
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs Specific Chainquery Jobs",
	Long:  `Allows for running different chainquery jobs without having to run the rest of the application`,
	Run: func(cmd *cobra.Command, args []string) {
		if viper.GetBool("codeprofile") {
			defer profile.Start(profile.NoShutdownHook).Stop()
		}
		job, ok := jobsMap[args[0]]
		if !ok {
			var jobs []string
			for job := range jobsMap {
				jobs = append(jobs, job)
			}
			logrus.Infof("Incorrect usage, should be: run <jobname>. Possible jobs are: %s", strings.Join(jobs, ", "))
			return
		}
		lbrycrdClient := lbrycrd.Init()
		defer lbrycrdClient.Shutdown()
		//Main Chainquery DB connection
		dbInstance, err := db.Init(config.GetMySQLDSN(), config.GetDebugQueryMode())
		if err != nil {
			log.Panic(err)
		}
		job()

		db.CloseDB(dbInstance)
	},
}
