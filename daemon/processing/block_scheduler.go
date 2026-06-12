package processing

import (
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"sort"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/metrics"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/chainquery/util"
	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/stop"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null/v8"
)

const (
	dependencyReasonUTXO      = "utxo"
	dependencyReasonClaim     = "claim"
	mysqlErrorLockWaitTimeout = 1205
	mysqlErrorDeadlock        = 1213
)

var (
	fetchRawTransaction      = lbrycrd.GetRawTransactionResponse
	cleanupBlockTransactions = cleanupAbortedBlockTransactions
	txRetryBackoff           = 10 * time.Millisecond
)

type txFetchJob struct {
	txID  string
	index int
}

type txFetchResult struct {
	tx    *lbrycrd.TxRawResult
	err   error
	index int
}

type blockTxDependency struct {
	parent string
	child  string
	reason string
}

type blockTxGraph struct {
	txByID     map[string]*lbrycrd.TxRawResult
	children   map[string][]blockTxDependency
	unresolved map[string]int
	completed  map[string]bool
	attempts   map[string]int
	edgeSet    map[blockTxDependency]struct{}
	order      []string
	ready      []string
}

type claimGraphEvent struct {
	claimID string
	writer  bool
	reader  bool
}

func syncTransactionsDependencyAware(stopper *stop.Group, txIDs []string, blockTime uint64, blockHeight uint64) error {
	if stopper == nil {
		stopper = stop.New(nil)
	}
	orderedTxs, txByID, err := fetchBlockRawTransactions(stopper, txIDs)
	if err != nil {
		return err
	}
	graph := newBlockTxGraph(orderedTxs, txByID)
	err = graph.buildDependencies()
	if err != nil {
		return err
	}
	return runBlockTxScheduler(stopper, graph, blockTime, blockHeight)
}

func fetchBlockRawTransactions(stopper *stop.Group, txIDs []string) ([]*lbrycrd.TxRawResult, map[string]*lbrycrd.TxRawResult, error) {
	orderedTxs := make([]*lbrycrd.TxRawResult, len(txIDs))
	txByID := make(map[string]*lbrycrd.TxRawResult, len(txIDs))
	if len(txIDs) == 0 {
		return orderedTxs, txByID, nil
	}

	workerCount := util.Min(len(txIDs), MaxParallelTxProcessing)
	if workerCount < 1 {
		workerCount = 1
	}

	fetchStopper := stop.New(stopper)
	jobs := make(chan txFetchJob)
	results := make(chan txFetchResult)
	for i := 0; i < workerCount; i++ {
		fetchStopper.Add(1)
		go runTxFetchWorker(fetchStopper, jobs, results)
	}
	fetchStopper.Add(1)
	go queueTxFetchJobs(fetchStopper, txIDs, jobs)

	for remaining := len(txIDs); remaining > 0; {
		select {
		case result := <-results:
			if result.err != nil {
				fetchStopper.Stop()
				fetchStopper.StopAndWait()
				return nil, nil, result.err
			}
			orderedTxs[result.index] = result.tx
			txByID[result.tx.Txid] = result.tx
			remaining--
		case <-stopper.Ch():
			fetchStopper.Stop()
			fetchStopper.StopAndWait()
			return nil, nil, ManualShutDownError
		}
	}
	fetchStopper.Stop()
	fetchStopper.StopAndWait()
	return orderedTxs, txByID, nil
}

func queueTxFetchJobs(stopper *stop.Group, txIDs []string, jobs chan<- txFetchJob) {
	defer stopper.Done()
	defer close(jobs)
	for i, txID := range txIDs {
		select {
		case jobs <- txFetchJob{txID: txID, index: i}:
		case <-stopper.Ch():
			return
		}
	}
}

func runTxFetchWorker(stopper *stop.Group, jobs <-chan txFetchJob, results chan<- txFetchResult) {
	defer stopper.Done()
	for {
		select {
		case <-stopper.Ch():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			tx, err := fetchRawTransaction(job.txID)
			select {
			case results <- txFetchResult{tx: tx, index: job.index, err: err}:
			case <-stopper.Ch():
				return
			}
		}
	}
}

func newBlockTxGraph(orderedTxs []*lbrycrd.TxRawResult, txByID map[string]*lbrycrd.TxRawResult) *blockTxGraph {
	graph := &blockTxGraph{
		txByID:     txByID,
		children:   make(map[string][]blockTxDependency, len(txByID)),
		unresolved: make(map[string]int, len(txByID)),
		completed:  make(map[string]bool, len(txByID)),
		attempts:   make(map[string]int, len(txByID)),
		edgeSet:    make(map[blockTxDependency]struct{}),
		order:      make([]string, 0, len(orderedTxs)),
		ready:      make([]string, 0, len(orderedTxs)),
	}
	for _, tx := range orderedTxs {
		if tx == nil {
			continue
		}
		graph.order = append(graph.order, tx.Txid)
		graph.unresolved[tx.Txid] = 0
	}
	return graph
}

func (graph *blockTxGraph) buildDependencies() error {
	graph.addUTXODependencies()
	graph.addClaimDependencies()
	graph.sortChildren()
	for _, txID := range graph.order {
		if graph.unresolved[txID] == 0 {
			graph.ready = append(graph.ready, txID)
			metrics.ProcessingSchedulerEvents.WithLabelValues("ready").Inc()
		}
	}
	if len(graph.ready) == 0 && len(graph.order) > 0 {
		return fmt.Errorf("%w: no ready transactions", ErrDependencyGraphStalled)
	}
	return nil
}

func (graph *blockTxGraph) addUTXODependencies() {
	for _, txID := range graph.order {
		tx := graph.txByID[txID]
		for _, vin := range tx.Vin {
			if _, ok := graph.txByID[vin.TxID]; ok {
				graph.addDependency(vin.TxID, txID, dependencyReasonUTXO)
			}
		}
	}
}

func (graph *blockTxGraph) addClaimDependencies() {
	latestWriter := make(map[string]string)
	waitingReaders := make(map[string][]string)
	for _, txID := range graph.order {
		tx := graph.txByID[txID]
		for _, event := range claimGraphEvents(tx) {
			if event.writer {
				if previousWriter := latestWriter[event.claimID]; previousWriter != "" {
					graph.addDependency(previousWriter, txID, dependencyReasonClaim)
				}
				for _, readerTxID := range waitingReaders[event.claimID] {
					graph.addDependency(readerTxID, txID, dependencyReasonClaim)
				}
				delete(waitingReaders, event.claimID)
				latestWriter[event.claimID] = txID
				continue
			}
			if event.reader {
				if previousWriter := latestWriter[event.claimID]; previousWriter != "" {
					graph.addDependency(previousWriter, txID, dependencyReasonClaim)
				}
				waitingReaders[event.claimID] = append(waitingReaders[event.claimID], txID)
			}
		}
	}
}

func (graph *blockTxGraph) addDependency(parent string, child string, reason string) {
	if parent == "" || child == "" || parent == child {
		return
	}
	edge := blockTxDependency{parent: parent, child: child, reason: reason}
	if _, ok := graph.edgeSet[edge]; ok {
		return
	}
	graph.edgeSet[edge] = struct{}{}
	graph.children[parent] = append(graph.children[parent], edge)
	graph.unresolved[child]++
	metrics.ProcessingSchedulerDependencyEdges.WithLabelValues(reason).Inc()
}

func (graph *blockTxGraph) sortChildren() {
	orderIndex := make(map[string]int, len(graph.order))
	for i, txID := range graph.order {
		orderIndex[txID] = i
	}
	for parent := range graph.children {
		sort.SliceStable(graph.children[parent], func(i int, j int) bool {
			left := graph.children[parent][i]
			right := graph.children[parent][j]
			if orderIndex[left.child] != orderIndex[right.child] {
				return orderIndex[left.child] < orderIndex[right.child]
			}
			return left.reason < right.reason
		})
	}
}

func claimGraphEvents(tx *lbrycrd.TxRawResult) []claimGraphEvent {
	events := make([]claimGraphEvent, 0)
	for _, vout := range tx.Vout {
		events = append(events, claimGraphEventsForVout(tx, vout)...)
	}
	return events
}

func claimGraphEventsForVout(tx *lbrycrd.TxRawResult, vout lbrycrd.Vout) (events []claimGraphEvent) {
	defer func() {
		if recovered := recover(); recovered != nil {
			logrus.Debugf("claim graph parse panic for tx %s vout %d: %v", tx.Txid, vout.N, recovered)
			events = nil
		}
	}()
	script, err := hex.DecodeString(vout.ScriptPubKey.Hex)
	if err != nil || len(script) == 0 {
		if err != nil {
			logrus.Debugf("claim graph script decode failed for tx %s vout %d: %s", tx.Txid, vout.N, err.Error())
		}
		return nil
	}
	if lbrycrd.IsClaimNameScript(script) {
		_, _, _, err := lbrycrd.ParseClaimNameScript(script)
		if err != nil {
			logrus.Debugf("claim graph claim-name parse skipped for tx %s vout %d: %s", tx.Txid, vout.N, err.Error())
			return nil
		}
		claimID, err := lbrycrd.ClaimIDFromOutpoint(tx.Txid, int(vout.N))
		if err == nil {
			return []claimGraphEvent{{claimID: claimID, writer: true}}
		}
		logrus.Debugf("claim graph claim-name parse skipped for tx %s vout %d: %s", tx.Txid, vout.N, err.Error())
		return nil
	}
	if lbrycrd.IsClaimUpdateScript(script) {
		_, claimID, _, _, err := lbrycrd.ParseClaimUpdateScript(script)
		if err == nil {
			return []claimGraphEvent{{claimID: claimID, writer: true}}
		}
		logrus.Debugf("claim graph update parse skipped for tx %s vout %d: %s", tx.Txid, vout.N, err.Error())
		return nil
	}
	if lbrycrd.IsClaimSupportScript(script) {
		_, claimID, _, _, err := lbrycrd.ParseClaimSupportScript(script)
		if err == nil {
			return []claimGraphEvent{{claimID: claimID, reader: true}}
		}
		logrus.Debugf("claim graph support parse skipped for tx %s vout %d: %s", tx.Txid, vout.N, err.Error())
		return nil
	}
	if lbrycrd.IsPurchaseScript(script) {
		purchase, err := lbrycrd.ParsePurchaseScript(script)
		if err == nil {
			claimID := hex.EncodeToString(util.ReverseBytes(purchase.GetClaimHash()))
			return []claimGraphEvent{{claimID: claimID, reader: true}}
		}
		logrus.Debugf("claim graph purchase parse skipped for tx %s vout %d: %s", tx.Txid, vout.N, err.Error())
	}
	return nil
}

func runBlockTxScheduler(stopper *stop.Group, graph *blockTxGraph, blockTime uint64, blockHeight uint64) error {
	if len(graph.order) == 0 {
		return nil
	}
	workerCount := util.Min(len(graph.order), MaxParallelTxProcessing)
	if workerCount < 1 {
		workerCount = 1
	}
	workerStopper := stop.New(stopper)
	retryStopper := stop.New(stopper)
	jobs := make(chan txToProcess)
	results := make(chan txProcessResult)
	retryReady := make(chan string)
	for i := 0; i < workerCount; i++ {
		workerStopper.Add(1)
		go runScheduledTxWorker(workerStopper, jobs, results)
	}

	active := 0
	completed := 0
	pendingRetries := 0
	var terminalErr error
	for completed < len(graph.order) {
		if terminalErr == nil {
			dispatched, err := dispatchReadyTransactions(stopper, graph, jobs, blockTime, blockHeight, workerCount-active)
			if err != nil {
				terminalErr = err
			}
			active += dispatched
		}
		if terminalErr != nil && active == 0 {
			break
		}
		if terminalErr != nil {
			<-results
			active--
			continue
		}
		if active == 0 && len(graph.ready) == 0 && pendingRetries == 0 {
			terminalErr = fmt.Errorf("%w: %d completed of %d transactions", ErrDependencyGraphStalled, completed, len(graph.order))
			break
		}
		select {
		case result := <-results:
			active--
			select {
			case <-stopper.Ch():
				terminalErr = ManualShutDownError
				metrics.ProcessingSchedulerEvents.WithLabelValues("cancellation").Inc()
				continue
			default:
			}
			if result.err != nil {
				err := handleScheduledTxError(result, graph, retryReady, retryStopper)
				if err != nil {
					metrics.ProcessingSchedulerEvents.WithLabelValues("terminal_failure").Inc()
					terminalErr = err
				} else {
					pendingRetries++
				}
				continue
			}
			completed++
			graph.completed[result.tx.Txid] = true
			graph.releaseChildren(result.tx.Txid)
		case txID := <-retryReady:
			pendingRetries--
			graph.ready = append(graph.ready, txID)
			metrics.ProcessingSchedulerEvents.WithLabelValues("ready").Inc()
		case <-stopper.Ch():
			terminalErr = ManualShutDownError
			metrics.ProcessingSchedulerEvents.WithLabelValues("cancellation").Inc()
		}
	}
	retryStopper.Stop()
	retryStopper.StopAndWait()
	workerStopper.Stop()
	workerStopper.StopAndWait()
	if terminalErr != nil {
		cleanupErr := cleanupBlockTransactions(graph.order)
		if cleanupErr != nil {
			return errors.Prefix(terminalErr.Error(), cleanupErr)
		}
		return terminalErr
	}
	return nil
}

func runScheduledTxWorker(stopper *stop.Group, jobs <-chan txToProcess, results chan<- txProcessResult) {
	defer stopper.Done()
	for {
		select {
		case <-stopper.Ch():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			err := processTx(job.tx, job.blockTime, job.blockHeight)
			if err != nil {
				metrics.ProcessingFailures.WithLabelValues("transaction").Inc()
				logrus.Debugf("processing tx failed %d times %s: %s", job.failcount+1, job.tx.Txid, err.Error())
			} else if job.failcount > 0 {
				logrus.Debugf("processing tx success after %d times %s", job.failcount, job.tx.Txid)
			}
			result := txProcessResult{
				tx:          job.tx,
				blockTime:   job.blockTime,
				blockHeight: job.blockHeight,
				err:         err,
				failcount:   job.failcount + 1,
			}
			results <- result
		}
	}
}

func dispatchReadyTransactions(stopper *stop.Group, graph *blockTxGraph, jobs chan<- txToProcess, blockTime uint64, blockHeight uint64, capacity int) (int, error) {
	dispatched := 0
	for capacity > 0 && len(graph.ready) > 0 {
		select {
		case <-stopper.Ch():
			return dispatched, ManualShutDownError
		default:
		}
		txID := graph.ready[0]
		graph.ready = graph.ready[1:]
		job := txToProcess{
			tx:          graph.txByID[txID],
			blockTime:   blockTime,
			blockHeight: blockHeight,
			failcount:   graph.attempts[txID],
		}
		select {
		case jobs <- job:
			dispatched++
			capacity--
		case <-stopper.Ch():
			return dispatched, ManualShutDownError
		}
	}
	return dispatched, nil
}

func (graph *blockTxGraph) releaseChildren(txID string) {
	for _, edge := range graph.children[txID] {
		graph.unresolved[edge.child]--
		if graph.unresolved[edge.child] == 0 {
			graph.ready = append(graph.ready, edge.child)
			metrics.ProcessingSchedulerEvents.WithLabelValues("ready").Inc()
		}
	}
}

func handleScheduledTxError(result txProcessResult, graph *blockTxGraph, retryReady chan<- string, retryStopper *stop.Group) error {
	graph.attempts[result.tx.Txid] = result.failcount
	missing, hasMissing := missingSourceOutputFromError(result.err)
	if hasMissing {
		if _, ok := graph.txByID[missing.PrevoutTxID]; ok {
			return fmt.Errorf("%w: same-block source output %s:%d missing while processing %s at block %d", ErrSchedulerInvariant, missing.PrevoutTxID, missing.PrevoutN, result.tx.Txid, result.blockHeight)
		}
		return result.err
	}
	if !isRetryableScheduledTxError(result.err) {
		return result.err
	}
	if result.failcount > MaxFailures {
		return errors.Prefix(fmt.Sprintf("transaction %s failed more than %d times", result.tx.Txid, MaxFailures), result.err)
	}
	metrics.ProcessingSchedulerEvents.WithLabelValues("retryable_failure").Inc()
	retryStopper.Add(1)
	go scheduleTxRetry(retryStopper, result.tx.Txid, txRetryBackoff, retryReady)
	return nil
}

func scheduleTxRetry(stopper *stop.Group, txID string, delay time.Duration, retryReady chan<- string) {
	defer stopper.Done()
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		select {
		case retryReady <- txID:
		case <-stopper.Ch():
		}
	case <-stopper.Ch():
	}
}

func cleanupAbortedBlockTransactions(txIDs []string) error {
	if len(txIDs) == 0 {
		return nil
	}
	inputs, err := model.Inputs(model.InputWhere.TransactionHash.IN(txIDs)).AllG()
	if err != nil {
		return errors.Err(err)
	}
	inputIDs := make([]uint64, 0, len(inputs))
	for _, input := range inputs {
		inputIDs = append(inputIDs, input.ID)
	}
	if len(inputIDs) == 0 {
		return nil
	}
	err = model.Outputs(model.OutputWhere.SpentByInputID.IN(inputIDs)).UpdateAllG(model.M{
		model.OutputColumns.IsSpent:        false,
		model.OutputColumns.SpentByInputID: null.Uint64{},
	})
	if err != nil {
		return errors.Err(err)
	}
	count, err := model.Outputs(model.OutputWhere.SpentByInputID.IN(inputIDs)).CountG()
	if err != nil {
		return errors.Err(err)
	}
	if count != 0 {
		return errors.Err("failed to reset %d spent outputs for aborted block", count)
	}
	return nil
}

func isRetryableStorageError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if stderrors.As(err, &mysqlErr) {
		return mysqlErr.Number == mysqlErrorLockWaitTimeout || mysqlErr.Number == mysqlErrorDeadlock
	}
	return false
}

func isRetryableScheduledTxError(err error) bool {
	return isRetryableStorageError(err)
}
