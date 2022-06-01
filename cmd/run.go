package cmd

import (
	"log"
	"strings"
	"time"

	"github.com/lbryio/chainquery/config"
	"github.com/lbryio/chainquery/daemon/jobs"
	"github.com/lbryio/chainquery/db"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/lbry.go/extras/errors"

	"github.com/pkg/profile"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var jobsMap = map[string]func(){
	"claimcount": func() {
		err := jobs.SyncClaimCntInChannel()
		if err != nil {
			logrus.Errorf("failure in running claimcount job: %s", errors.FullTrace(err))
		}
	},
	"claimtrie":        jobs.ClaimTrieSync,
	"certificate":      jobs.CertificateSync,
	"mempool":          jobs.MempoolSync,
	"transactionvalue": jobs.TransactionValueSync,
	"chain":            jobs.ChainSync,
	"outputfix":        jobs.OutputFixSync,
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
		logrus.Debugf("Starting job '%s'", args[0])
		start := time.Now()
		job()
		logrus.Debugf("Finished job '%s', it took %s", args[0], time.Since(start))

		db.CloseDB(dbInstance)
	},
}
