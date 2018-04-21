package daemon

import (
	"runtime"
	"time"

	"github.com/lbryio/chainquery/daemon/jobs"
	"github.com/lbryio/chainquery/daemon/processing"
	"github.com/lbryio/chainquery/daemon/upgrademanager"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"

	log "github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

const (
	// Mode for processing speed
	BEASTMODE      = 0 // serialized until finished - new thread each daemon iteration.
	SLOWSTEADYMODE = 1 // 1 block per 100 ms
	DELAYMODE      = 2 // 1 block per delay
	DAEMONMODE     = 3 // 1 block per Daemon iteration
)

var workers = runtime.NumCPU() / 2 //Split cores between processors and lbycrd
var lastHeightProcessed uint64 = 0 // Around 165,000 is when protobuf takes affect.
var blockHeight uint64 = 0
var running = false
var Reindex = false
var BlockConfirmationBuffer uint64 = 6 //Block is accepted at 6 confirmations

//Configuration
var ProcessingMode int            //Set in main on init
var ProcessingDelay time.Duration //Set by `applySettings`
var DaemonDelay time.Duration     //Set by `applySettings`
var iteration int64 = 0

func DoYourThing() {
	go initJobs()
	upgrademanager.RunUpgradesForVersion()
	runDaemon()
}

func initJobs() {
	jobs.ClaimTrieSync()
	t := time.NewTicker(5 * time.Minute)
	for {
		<-t.C
		jobs.ClaimTrieSync()
	}
}

func runDaemon() {
	lastBlock, _ := model.Blocks(boil.GetDB(), qm.OrderBy(model.BlockColumns.Height+" DESC"), qm.Limit(1)).One()
	if lastBlock != nil && lastBlock.Height > 100 && !Reindex {
		lastHeightProcessed = lastBlock.Height - 100 //Start 100 sooner just in case something happened.
	}
	log.Info("Daemon initialized and running")
	for {
		time.Sleep(DaemonDelay)
		if !running {
			running = true
			log.Info("Running daemon iteration ", iteration)
			go daemonIteration()
			iteration++
		}
	}
}

func daemonIteration() error {

	height, err := lbrycrd.GetBlockCount()
	if err != nil {
		return err
	}
	blockHeight = *height - BlockConfirmationBuffer
	if lastHeightProcessed == uint64(0) {
		processing.RunBlockProcessing(&lastHeightProcessed)
	}
	for {
		next := lastHeightProcessed + 1
		if blockHeight >= next {
			processing.RunBlockProcessing(&next)
			lastHeightProcessed = next
		}
		if next%50 == 0 {
			log.Info("running iteration at block height ", next, runtime.NumGoroutine(), " go routines")
		}
		workToDo := lastHeightProcessed+uint64(1) < blockHeight && lastHeightProcessed != 0
		if workToDo {
			time.Sleep(ProcessingDelay)
			continue
		} else if *height != 0 {
			running = false
			break
		}
	}
	return nil
}

func ApplySettings(processingDelay time.Duration, daemonDelay time.Duration) {
	DaemonDelay = daemonDelay
	ProcessingDelay = processingDelay
	if ProcessingMode == BEASTMODE {
		ProcessingDelay = 0 * time.Millisecond
	} else if ProcessingMode == SLOWSTEADYMODE {
		ProcessingDelay = 100 * time.Millisecond
	} else if ProcessingMode == DELAYMODE {
		ProcessingDelay = processingDelay
	} else if ProcessingMode == DAEMONMODE {
		ProcessingDelay = daemonDelay //
	}
}
