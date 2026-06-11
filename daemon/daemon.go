package daemon

import (
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"sync"
	"sync/atomic"
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

	legacyBlockStateBackfillInterval = 15 * time.Minute
)

var lastHeightProcessed uint64 // Around 165,000 is when protobuf takes affect.
var lastHeightLogged uint64
var blockHeight uint64
var running atomic.Bool
var reindex = false

// Configuration
var processingMode int                          //Set by `applySettings`
var processingDelay time.Duration               //Set by `applySettings`
var daemonDelay time.Duration                   //Set by `applySettings`
var blockProcessingTimeout = 10 * time.Minute   //Set by `applySettings`
var blockProcessingDumpDelay = 10 * time.Minute //Set by `applySettings`
var exitOnBlockProcessingTimeout = false        //Set by `applySettings`
var blockWorkers uint64 = 1                     //ToDo Should be configurable
var iteration int64
var jobsInitialized atomic.Bool

var blockQueue = make(chan uint64)
var blockProcessedChan = make(chan uint64)
var stopper = stop.New()
var watchdogState blockWatchdogState

type blockWatchdogState struct {
	mutex             sync.Mutex
	lastSuccessAt     time.Time
	inflightStartedAt time.Time
	lastDumpAt        time.Time
	lastSuccessHeight uint64
	targetHeight      uint64
	inflightHeight    uint64
}

type blockWatchdogSnapshot struct {
	lastSuccessAt     time.Time
	inflightStartedAt time.Time
	lastSuccessHeight uint64
	targetHeight      uint64
	inflightHeight    uint64
	elapsed           time.Duration
}

// DoYourThing kicks off the daemon and jobs
func DoYourThing() {

	upgrademanager.RunUpgradesForVersion()
	err := processing.CleanupIncompleteHead()
	if err != nil {
		log.Fatal(errors.Prefix("could not clean up incomplete head block", err))
	}
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
	scheduleJob(backfillLegacyBlockStates, "Legacy Block State Backfill", legacyBlockStateBackfillInterval)
}

func backfillLegacyBlockStates() {
	count, err := processing.BackfillLegacyBlockStates(processing.LegacyBlockBackfillBatchSize)
	if err != nil {
		log.Error(errors.Prefix("could not backfill legacy block states", err))
		return
	}
	if count > 0 {
		log.Infof("backfilled processing state for %d legacy blocks", count)
	}
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
	startBlockProcessingWatchdog()
	lastBlock, _ := model.Blocks(qm.OrderBy(model.BlockColumns.Height+" DESC"), qm.Limit(1)).OneG()
	if lastBlock != nil && !reindex {
		//Always
		lastHeightProcessed = lastBlock.Height
		recordBlockSuccess(lastHeightProcessed)
	}
	log.Info("Daemon initialized and running")
	t := time.NewTicker(daemonDelay)
	defer t.Stop()
	for {
		select {
		case <-stopper.Ch():
			log.Info("stopping daemon...")
			return
		case <-t.C:
			if running.CompareAndSwap(false, true) {
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
		log.Error(errors.Prefix("Could not get block height", err))
		running.Store(false)
		return
	}
	blockHeight = *height
	recordDaemonTarget(blockHeight)
	if lastHeightProcessed == uint64(0) {
		processedHeight, ok := processBlockHeight(lastHeightProcessed)
		if !ok {
			return
		}
		lastHeightProcessed = processedHeight
	}
	for {
		select {
		case <-stopper.Ch():
			log.Info("stopping daemon iteration...")
			return
		default:
			next := lastHeightProcessed + 1
			if blockHeight >= next {
				processedHeight, ok := processBlockHeight(next)
				if !ok {
					return
				}
				lastHeightProcessed = processedHeight
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
			running.Store(false)
			if jobsInitialized.CompareAndSwap(false, true) {
				asyncStoppable(initJobs)
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
	blockProcessingTimeout = settings.BlockProcessingTimeout
	blockProcessingDumpDelay = settings.BlockProcessingDumpInterval
	exitOnBlockProcessingTimeout = settings.ExitOnBlockProcessingTimeout
	if daemonDelay <= 0 {
		log.Warn("daemon delay must be greater than zero; using 1s")
		daemonDelay = time.Second
	}
	if blockProcessingDumpDelay <= 0 {
		blockProcessingDumpDelay = blockProcessingTimeout
	}

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
		go runBlockProcessor(i, jobs)
	}
}

func runBlockProcessor(worker int, jobs <-chan uint64) {
	defer stopper.DoneNamed("block worker " + strconv.Itoa(worker))
	log.Info("block worker ", worker+1, " running")
	BlockProcessor(jobs, worker)
}

// BlockProcessor takes a channel of block heights to process. When a new one comes in it runs block processing for
// the block height
func BlockProcessor(blocks <-chan uint64, worker int) {
	for {
		select {
		case <-stopper.Ch():
			return
		case block, ok := <-blocks:
			if !ok {
				return
			}
			processedHeight := processBlockWithRecover(block)
			select {
			case blockProcessedChan <- processedHeight:
			case <-stopper.Ch():
				return
			}
		}
	}
}

func processBlockWithRecover(height uint64) (processedHeight uint64) {
	processedHeight = rollbackBlockHeight(height)
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		log.Errorf("block processor panic at height %d: %v", height, recovered)
		log.Error(string(debug.Stack()))
		err := processing.MarkIncompleteBlockHeight(height)
		if err != nil {
			log.Error(errors.Prefix("could not mark panicked block incomplete", err))
		}
		err = processing.CleanupIncompleteHead()
		if err != nil {
			log.Error(errors.Prefix("could not clean up panicked incomplete head", err))
		}
	}()
	return processing.RunBlockProcessing(stopper, height)
}

func rollbackBlockHeight(height uint64) uint64 {
	if height == 0 {
		return 0
	}
	return height - 1
}

func processBlockHeight(height uint64) (uint64, bool) {
	recordBlockStart(height)
	if !sendBlockHeight(height) {
		clearInFlightBlock(height)
		return 0, false
	}
	processedHeight, ok := waitForBlockProcessed(height)
	if ok {
		recordBlockFinished(height, processedHeight)
	} else {
		clearInFlightBlock(height)
	}
	return processedHeight, ok
}

func sendBlockHeight(height uint64) bool {
	if blockProcessingTimeout <= 0 {
		select {
		case blockQueue <- height:
			return true
		case <-stopper.Ch():
			return false
		}
	}

	timer := time.NewTimer(blockProcessingTimeout)
	defer timer.Stop()
	for {
		select {
		case blockQueue <- height:
			return true
		case <-stopper.Ch():
			return false
		case <-timer.C:
			handleBlockProcessingTimeout("block queue send timeout", height, true)
			return false
		}
	}
}

func waitForBlockProcessed(height uint64) (uint64, bool) {
	if blockProcessingTimeout <= 0 {
		select {
		case processedHeight := <-blockProcessedChan:
			return processedHeight, true
		case <-stopper.Ch():
			return 0, false
		}
	}

	timer := time.NewTimer(blockProcessingTimeout)
	defer timer.Stop()

	for {
		select {
		case processedHeight := <-blockProcessedChan:
			return processedHeight, true
		case <-stopper.Ch():
			return 0, false
		case <-timer.C:
			handleBlockProcessingTimeout("block processing handshake timeout", height, true)
			timer.Reset(nextBlockProcessingDumpDelay())
		}
	}
}

func handleBlockProcessingTimeout(reason string, height uint64, force bool) {
	snapshot, ok := recordBlockTimeoutDump(height, force)
	if !ok {
		return
	}
	log.Errorf("%s for in-flight block %d after %s; target=%d last_success=%d last_success_at=%s", reason, snapshot.inflightHeight, snapshot.elapsed, snapshot.targetHeight, snapshot.lastSuccessHeight, snapshot.lastSuccessAt.Format(time.RFC3339))
	dumpGoroutines(reason)
	if exitOnBlockProcessingTimeout {
		os.Exit(1)
	}
	log.Errorf("daemon marked unhealthy; waiting for in-flight block %d to finish before queueing more work", height)
}

func nextBlockProcessingDumpDelay() time.Duration {
	if blockProcessingDumpDelay > 0 {
		return blockProcessingDumpDelay
	}
	return blockProcessingTimeout
}

func startBlockProcessingWatchdog() {
	stopper.AddNamed(1, "block processing watchdog")
	go runBlockProcessingWatchdog()
}

func runBlockProcessingWatchdog() {
	defer stopper.DoneNamed("block processing watchdog")
	interval := blockWatchdogInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-stopper.Ch():
			return
		case <-ticker.C:
			snapshot, ok := blockTimeoutSnapshot(false)
			if !ok {
				continue
			}
			log.Errorf("daemon watchdog observed stalled in-flight block %d after %s; target=%d last_success=%d last_success_at=%s", snapshot.inflightHeight, snapshot.elapsed, snapshot.targetHeight, snapshot.lastSuccessHeight, snapshot.lastSuccessAt.Format(time.RFC3339))
			dumpGoroutines("daemon block processing watchdog")
			if exitOnBlockProcessingTimeout {
				os.Exit(1)
			}
		}
	}
}

func blockWatchdogInterval() time.Duration {
	if blockProcessingTimeout > 0 && blockProcessingTimeout < time.Minute {
		return blockProcessingTimeout
	}
	return time.Minute
}

func recordDaemonTarget(height uint64) {
	watchdogState.mutex.Lock()
	defer watchdogState.mutex.Unlock()
	watchdogState.targetHeight = height
	if watchdogState.lastSuccessAt.IsZero() {
		watchdogState.lastSuccessAt = time.Now()
		watchdogState.lastSuccessHeight = lastHeightProcessed
	}
}

func recordBlockStart(height uint64) {
	watchdogState.mutex.Lock()
	defer watchdogState.mutex.Unlock()
	watchdogState.inflightHeight = height
	watchdogState.inflightStartedAt = time.Now()
}

func recordBlockSuccess(height uint64) {
	watchdogState.mutex.Lock()
	defer watchdogState.mutex.Unlock()
	watchdogState.lastSuccessHeight = height
	watchdogState.lastSuccessAt = time.Now()
}

func recordBlockFinished(height uint64, processedHeight uint64) {
	watchdogState.mutex.Lock()
	defer watchdogState.mutex.Unlock()
	if watchdogState.inflightHeight == height {
		watchdogState.inflightHeight = 0
		watchdogState.inflightStartedAt = time.Time{}
	}
	watchdogState.lastSuccessHeight = processedHeight
	watchdogState.lastSuccessAt = time.Now()
}

func clearInFlightBlock(height uint64) {
	watchdogState.mutex.Lock()
	defer watchdogState.mutex.Unlock()
	if watchdogState.inflightHeight == height {
		watchdogState.inflightHeight = 0
		watchdogState.inflightStartedAt = time.Time{}
	}
}

func recordBlockTimeoutDump(height uint64, force bool) (blockWatchdogSnapshot, bool) {
	return blockTimeoutSnapshot(force)
}

func blockTimeoutSnapshot(force bool) (blockWatchdogSnapshot, bool) {
	watchdogState.mutex.Lock()
	defer watchdogState.mutex.Unlock()
	if blockProcessingTimeout <= 0 || watchdogState.inflightStartedAt.IsZero() {
		return blockWatchdogSnapshot{}, false
	}
	now := time.Now()
	elapsed := now.Sub(watchdogState.inflightStartedAt)
	if elapsed < blockProcessingTimeout {
		return blockWatchdogSnapshot{}, false
	}
	if !force && !watchdogState.lastDumpAt.IsZero() && now.Sub(watchdogState.lastDumpAt) < nextBlockProcessingDumpDelay() {
		return blockWatchdogSnapshot{}, false
	}
	watchdogState.lastDumpAt = now
	return blockWatchdogSnapshot{
		inflightHeight:    watchdogState.inflightHeight,
		targetHeight:      watchdogState.targetHeight,
		lastSuccessHeight: watchdogState.lastSuccessHeight,
		lastSuccessAt:     watchdogState.lastSuccessAt,
		inflightStartedAt: watchdogState.inflightStartedAt,
		elapsed:           elapsed,
	}, true
}

func dumpGoroutines(reason string) {
	log.Error("dumping goroutine stacks: ", reason)
	profile := pprof.Lookup("goroutine")
	if profile == nil {
		log.Error("goroutine profile unavailable")
		return
	}
	err := profile.WriteTo(os.Stderr, 2)
	if err != nil {
		log.Error(errors.Prefix("could not write goroutine dump", err))
	}
}
