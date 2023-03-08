package cmd

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/lbryio/chainquery/config"
	"github.com/lbryio/chainquery/daemon/processing"
	"github.com/lbryio/chainquery/db"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/chainquery/util"

	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/pkg/profile"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(blockCmd)
}

var blockCmd = &cobra.Command{
	Use:   "block",
	Short: "Processes a specific block height",
	Long: `This is useful for testing locally. You can just sync a particular problematic block. 
			This will remove the block from the database before syncing it.`,
	Run: func(cmd *cobra.Command, args []string) {
		if viper.GetBool("codeprofile") {
			defer profile.Start(profile.NoShutdownHook).Stop()
		}
		blockHeight, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			logrus.Panic(errors.Prefix("Could not parse block height passed", err))
		}
		lbrycrdClient := lbrycrd.Init()
		defer lbrycrdClient.Shutdown()
		blockHash, err := lbrycrdClient.GetBlockHash(blockHeight)
		if err != nil {
			logrus.Panic(errors.Prefix(fmt.Sprintf("Could not get block hash @ height %d", blockHeight), err))
		}
		//Main Chainquery DB connection
		dbInstance, err := db.Init(config.GetMySQLDSN(), config.GetDebugQueryMode())
		if err != nil {
			logrus.Panic(err)
		}
		defer db.CloseDB(dbInstance)
		logrus.Infof("Running processor on block %d with hash %s", blockHeight, blockHash)
		block, err := model.Blocks(model.BlockWhere.Hash.EQ(blockHash.String())).OneG()
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			logrus.Panic(errors.Err(err))
		}
		if block != nil {
			err = block.DeleteG()
			if err != nil {
				logrus.Fatal(errors.Err(err))
			}
			logrus.Info("Block successfully removed")
		}
		jsonBlock, err := lbrycrd.GetBlock(blockHash.String())
		if err != nil {
			logrus.Fatal(errors.Err(err))
		}
		logrus.Info("processing block started")
		defer util.TimeTrack(time.Now(), "block processing", "always")
		_, err = processing.ProcessBlock(uint64(blockHeight), nil, jsonBlock)
		if err != nil {
			logrus.Fatal(errors.Err(err))
		}
		logrus.Info("processing block finished")
	},
}
