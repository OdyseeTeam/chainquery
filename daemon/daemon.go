package daemon

import (
	"runtime"
	"time"

	"github.com/lbryio/chainquery/daemon/jobs"
	"github.com/lbryio/chainquery/daemon/processing"
	"github.com/lbryio/chainquery/daemon/upgrademanager"
	"github.com/lbryio/chainquery/global"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"

	log "github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

const (
	// BEASTMODE serialized until finished - new thread each daemon iteration.
	beastmode = 0
	// SLOWSTEADYMODE 1 block per 100 ms
	slowsteadymode = 1
	// DELAYMODE(2) 1 block per delay
	// DAEMONMODE 1 block per Daemon iteration
	daemonmode = 3
)

var lastHeightProcessed uint64 // Around 165,000 is when protobuf takes affect.
var blockHeight uint64
var running = false
var reindex = false

var blockConfirmationBuffer uint64 = 6 //Block is accepted at 6 confirmations

//Configuration
var processingMode int            //Set by `applySettings`
var processingDelay time.Duration //Set by `applySettings`
var daemonDelay time.Duration     //Set by `applySettings`
var iteration int64

//DoYourThing kicks off the daemon and jobs
func DoYourThing() {
	go initJobs()
	upgrademanager.RunUpgradesForVersion()
	runDaemon()
}

func initJobs() {
	go jobs.ClaimTrieSync()
	t := time.NewTicker(15 * time.Minute)
	for {
		<-t.C
		go jobs.ClaimTrieSync()
	}
}

func runDaemon() {
	lastBlock, _ := model.Blocks(boil.GetDB(), qm.OrderBy(model.BlockColumns.Height+" DESC"), qm.Limit(1)).One()
	if lastBlock != nil && lastBlock.Height > 100 && !reindex {
		lastHeightProcessed = lastBlock.Height - 1 //Start 1 sooner just in case something happened.
	}
	log.Info("Daemon initialized and running")
	for {
		time.Sleep(daemonDelay)
		if !running {
			running = true
			log.Debug("Running daemon iteration ", iteration)
			go daemonIteration()
			iteration++
		}
	}
}

func daemonIteration() {

	height, err := lbrycrd.GetBlockCount()
	if err != nil {
		log.Error(err)
	}
	blockHeight = *height - blockConfirmationBuffer
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
			time.Sleep(processingDelay)
			continue
		} else if *height != 0 {
			running = false
			break
		}
	}
}

// ApplySettings sets the specific daemon settings from a configuration
func ApplySettings(settings global.DaemonSettings) {

	processingMode = settings.DaemonMode
	daemonDelay = settings.DaemonDelay
	processingDelay = settings.ProcessingDelay

	if processingMode == beastmode {
		processingDelay = 0 * time.Millisecond
	} else if processingMode == slowsteadymode {
		processingDelay = 100 * time.Millisecond
	} else if processingMode == daemonmode {
		processingDelay = daemonDelay //
	}
}
