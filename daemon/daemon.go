package daemon

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/lbryio/chainquery/daemon/jobs"
	"github.com/lbryio/chainquery/daemon/processing"
	"github.com/lbryio/chainquery/daemon/upgrademanager"
	"github.com/lbryio/chainquery/global"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/stop"

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
var lastHeightLogged uint64
var blockHeight uint64
var running = false
var reindex = false

//Configuration
var processingMode int            //Set by `applySettings`
var processingDelay time.Duration //Set by `applySettings`
var daemonDelay time.Duration     //Set by `applySettings`
var blockWorkers uint64 = 1       //ToDo Should be configurable
var iteration int64

var blockQueue = make(chan uint64)
var blockProcessedChan = make(chan uint64)
var stopper = stop.New()

//DoYourThing kicks off the daemon and jobs
func DoYourThing() {

	upgrademanager.RunUpgradesForVersion()
	asyncStoppable(initJobs)
	asyncStoppable(runDaemon)

	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)
	<-interruptChan
	shutdownDaemon()
}

func initJobs() {
	//ClaimTrieSync
	scheduleJob(jobs.ClaimTrieSync, 15*time.Minute)
	scheduleJob(jobs.MempoolSync, 1*time.Second)
}

func shutdownDaemon() {
	log.Info("Shutting down daemon...") //
	stopper.StopAndWait()
}

func scheduleJob(job func(), howOften time.Duration) {
	asyncStoppable(job)
	stopper.Add(1)
	go func() {
		defer stopper.Done()
		t := time.NewTicker(howOften)
		for {
			select {
			case <-stopper.Ch():
				log.Info("stopping job scheduler...")
				return
			case <-t.C:
				asyncStoppable(job)
			}
		}
	}()
}

func runDaemon() {
	initBlockWorkers(int(blockWorkers), blockQueue)
	lastBlock, _ := model.Blocks(boil.GetDB(), qm.OrderBy(model.BlockColumns.Height+" DESC"), qm.Limit(1)).One()
	if lastBlock != nil && !reindex {
		//Always
		lastHeightProcessed = lastBlock.Height
	}
	log.Info("Daemon initialized and running")
	for {
		select {
		case <-stopper.Ch():
			log.Info("stopping daemon...")
			return
		default:
			if !running {
				running = true
				log.Debug("Running daemon iteration ", iteration)
				asyncStoppable(daemonIteration)
				iteration++
			}
			time.Sleep(daemonDelay)
		}
	}
}

func asyncStoppable(function func()) {
	stopper.Add(1)
	go func() {
		stopper.Done()
		function()
	}()
}

func daemonIteration() {
	height, err := lbrycrd.GetBlockCount()
	if err != nil {
		log.Error(err)
	}
	blockHeight = *height
	if lastHeightProcessed == uint64(0) {
		blockQueue <- lastHeightProcessed
		lastHeightProcessed = <-blockProcessedChan
	}
	for {

		select {
		case <-stopper.Ch():
			log.Info("stopping daemon iteration...")
			return
		default:
			next := lastHeightProcessed + 1
			if blockHeight >= next {
				blockQueue <- next
				//Forces single threaded block processing
				lastHeightProcessed = <-blockProcessedChan
			}
			if next%50 == 0 && next != lastHeightLogged {
				log.Info("running iteration at block height ", next, runtime.NumGoroutine(), " go routines")
				lastHeightLogged = next
			}
			workToDo := lastHeightProcessed < blockHeight && lastHeightProcessed != 0
			if workToDo {
				time.Sleep(processingDelay)
				continue
			} else if *height != 0 {
				running = false
				return
			}
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

func initBlockWorkers(nrWorkers int, jobs <-chan uint64) {
	for i := 0; i < nrWorkers; i++ {
		stopper.Add(1)
		go func(worker int) {
			defer stopper.Done()
			log.Info("block worker ", worker+1, " running")
			BlockProcessor(jobs, worker)
		}(i)
	}
}

// BlockProcessor takes a channel of block heights to process. When a new one comes in it runs block processing for
// the block height
func BlockProcessor(blocks <-chan uint64, worker int) {
	for block := range blocks {
		blockProcessedChan <- processing.RunBlockProcessing(&block)
	}
}
