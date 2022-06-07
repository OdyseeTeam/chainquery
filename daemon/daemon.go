package daemon

import (
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/lbryio/chainquery/daemon/jobs"
	"github.com/lbryio/chainquery/daemon/processing"
	"github.com/lbryio/chainquery/daemon/upgrademanager"
	"github.com/lbryio/chainquery/global"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/stop"

	log "github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
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
var jobsInitialized = false

var blockQueue = make(chan uint64)
var blockProcessedChan = make(chan uint64)
var stopper = stop.New()

//DoYourThing kicks off the daemon and jobs
func DoYourThing() {

	upgrademanager.RunUpgradesForVersion()
	asyncStoppable(runDaemon)

	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)
	<-interruptChan
	ShutdownDaemon()
}

func initJobs() {
	scheduleJob(jobs.ClaimTrieSyncAsync, "Claimtrie Sync", 15*time.Minute)
	scheduleJob(jobs.MempoolSync, "Mempool Sync", 1*time.Second)
	scheduleJob(jobs.CertificateSync, "Certificate Sync", 5*time.Second)
	scheduleJob(jobs.ValidateChain, "Validate Chain", 24*time.Hour)
	scheduleJob(jobs.SyncAddressBalancesJob, "Address Balance Sync", 24*time.Hour)
	scheduleJob(jobs.TransactionValueASync, "Transaction Value Sync", 24*time.Hour)
	scheduleJob(jobs.SyncClaimsInChannelJob, "Claim Count in Channel Sync", 24*time.Hour)
	//ChainSync job should never be run later than 2.5 minutes or its possible it will never loop back due to coinbase time
	scheduleJob(jobs.ChainSyncAsync, "Chain Sync", 5*time.Second)
}

// ShutdownDaemon shuts the daemon down gracefully without corrupting the data.
func ShutdownDaemon() {
	log.Info("Shutting down daemon...") //
	stopper.StopAndWait()
}

func scheduleJob(job func(), name string, howOften time.Duration) {
	stopper.AddNamed(1, "scheduled job "+name)
	go func() {
		defer stopper.DoneNamed("scheduled job " + name)
		t := time.NewTicker(howOften)
		for {
			select {
			case <-stopper.Ch():
				log.Info("stopping scheduled job: ", name)
				return
			case <-t.C:
				asyncStoppable(job)
			}
		}
	}()
}

func runDaemon() {
	initBlockWorkers(int(blockWorkers), blockQueue)
	lastBlock, _ := model.Blocks(qm.OrderBy(model.BlockColumns.Height+" DESC"), qm.Limit(1)).OneG()
	if lastBlock != nil && !reindex {
		//Always
		lastHeightProcessed = lastBlock.Height
	}
	log.Info("Daemon initialized and running")
	t := time.NewTicker(daemonDelay)
	for {
		select {
		case <-stopper.Ch():
			log.Info("stopping daemon...")
			return
		case <-t.C:
			if !running {
				running = true
				log.Debug("Running daemon iteration ", iteration)
				asyncStoppable(daemonIteration)
				iteration++
			}
		}
	}
}

func asyncStoppable(function func()) {
	stopper.AddNamed(1, "stoppable - "+runtime.FuncForPC(reflect.ValueOf(function).Pointer()).Name())
	go func() {
		defer stopper.DoneNamed("stoppable - " + runtime.FuncForPC(reflect.ValueOf(function).Pointer()).Name())
		function()
	}()
}

func daemonIteration() {
	height, err := lbrycrd.GetBlockCount()
	if err != nil {
		log.Error(errors.Prefix("Could not get block height:", err))
		running = false
		return
	}
	blockHeight = *height
	if lastHeightProcessed == uint64(0) {
		blockQueue <- lastHeightProcessed
		lastHeightProcessed = <-blockProcessedChan
	}
	for {
		select {
		case <-stopper.Ch():
			close(blockQueue)
			log.Info("stopping daemon iteration...")
			close(blockProcessedChan)
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
			} else {
				log.Debug("running iteration at block height ", next, runtime.NumGoroutine(), " go routines")
			}
			workToDo := lastHeightProcessed < blockHeight && lastHeightProcessed != 0
			if workToDo {
				time.Sleep(processingDelay)
				continue
			}
			running = false
			if !jobsInitialized {
				asyncStoppable(initJobs)
				jobsInitialized = true
			}
			return
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
		stopper.AddNamed(1, "block worker "+strconv.Itoa(i))
		go func(worker int) {
			defer stopper.DoneNamed("block worker " + strconv.Itoa(worker))
			log.Info("block worker ", worker+1, " running")
			BlockProcessor(jobs, worker)
		}(i)
	}
}

// BlockProcessor takes a channel of block heights to process. When a new one comes in it runs block processing for
// the block height
func BlockProcessor(blocks <-chan uint64, worker int) {
	for {
		select {
		case <-stopper.Ch():
			return
		case block := <-blocks:
			blockProcessedChan <- processing.RunBlockProcessing(stopper, block)
		}
	}
}
