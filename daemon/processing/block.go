package processing

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/metrics"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/chainquery/sockety"
	"github.com/lbryio/chainquery/util"

	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/stop"

	"github.com/OdyseeTeam/sockety/socketyapi"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

// BlockLock is used to lock block processing to a single parent thread.
var BlockLock = sync.Mutex{}

// ManualShutDownError Error with special handling. Used to stop the concurrency pipeline for processing blocks midway.
var ManualShutDownError = errors.Err("Daemon stopped manually!")

const (
	BlockProcessingStateComplete   = "complete"
	BlockProcessingStateProcessing = "processing"
	BlockProcessingStateIncomplete = "incomplete"
	MempoolBlockHash               = "MEMPOOL"
	LegacyBlockBackfillBatchSize   = 100
	blockDeleteRetryAttempts       = 3
)

var blockDeleteRetryDelay = time.Second

// RunBlockProcessing runs the processing of a block at a specific height. While any height can be passed in it is
// important to note that if the previous block is not processed it will panic to prevent corruption because blocks
// must be processed in order.
func RunBlockProcessing(stopper *stop.Group, height uint64) uint64 {
	defer metrics.Processing(time.Now(), "block")
	defer util.TimeTrack(time.Now(), "runBlockProcessing", "daemonprofile")
	if height == 0 {
		err := processGenesisBlock()
		if err != nil {
			logrus.Fatal("Error processing Genesis block!", err)
		}
		return height
	}
	jsonBlock, err := getBlockToProcess(&height)
	if err != nil {
		logrus.Error("Get Block Error: ", err)
		//ToDo - Should just return error...that is for another day
		return height - 1
	}
	reorgHeight, err := checkHandleReorg(height, jsonBlock.PreviousBlockHash)
	if err != nil {
		logrus.Error("Reorg Handling Error: ", err)
		//ToDo - Should just return error...that is for another day
		return height - 1
	}
	if reorgHeight != height {
		return reorgHeight
	}

	//This is an important lock to make sure we don't concurrently save transaction inputs/outputs accidentally via the
	// mempool sync.
	BlockLock.Lock()
	defer BlockLock.Unlock()

	block, err := ProcessBlock(height, stopper, jsonBlock)
	if err != nil {
		metrics.ProcessingFailures.WithLabelValues("block").Inc()
		rollBackHeight := height - 1
		if block != nil {
			blockRemovalError := deleteBlockWithRetry(block)
			if blockRemovalError != nil {
				logrus.Errorf("could not delete block with bad data at height %d: %s", height, blockRemovalError.Error())
			}
		}
		if err.Error() == ManualShutDownError.Error() {
			return rollBackHeight
		}
		logrus.Error("Block Processing Error: ", errors.FullTrace(err))
		logrus.Warning("Ran into transaction sync error at height", height, ". Rolling block back to height", height-1)
		//ToDo - Should just return error...that is for another day
		return rollBackHeight
	}

	return height
}

// ProcessBlock processing a specific block and returns an error. Use this to process a block having a custom handling
// of the error.
func ProcessBlock(height uint64, stopper *stop.Group, jsonBlock *lbrycrd.GetBlockResponse) (*model.Block, error) {
	block, err := parseBlockInfo(height, jsonBlock)
	if err != nil {
		return nil, errors.Err(err)
	}
	err = setPreviousBlockInfo(height, jsonBlock.Hash)
	if err != nil {
		return block, errors.Err(err)
	}
	txs := jsonBlock.Tx
	err = syncTransactionsOfBlock(stopper, txs, block.BlockTime, block.Height)
	if err != nil {
		return block, errors.Err(err)
	}
	err = markBlockProcessingState(block, BlockProcessingStateComplete)
	if err != nil {
		return block, errors.Err(err)
	}
	sockety.SendNotification(socketyapi.SendNotificationArgs{
		Service: socketyapi.BlockChain,
		Type:    "new_block",
		IDs:     []string{"blocks", strconv.Itoa(int(height))},
		Data:    map[string]interface{}{"block": jsonBlock},
	})
	return block, nil
}

// setPreviousBlockInfo sets the NextBlockHash field from the previous block
func setPreviousBlockInfo(currentHeight uint64, currentBLockHash string) error {
	if currentHeight < 1 {
		return nil
	}
	prevBlock, err := model.Blocks(qm.Where(model.BlockColumns.Height+"=?", currentHeight-1)).OneG()
	if err != nil {
		return errors.Err(err)
	}
	prevBlock.NextBlockHash.SetValid(currentBLockHash)
	err = prevBlock.UpdateG(boil.Infer())
	return errors.Err(err)
}

func parseBlockInfo(blockHeight uint64, jsonBlock *lbrycrd.GetBlockResponse) (*model.Block, error) {
	block := &model.Block{}
	foundBlock, _ := model.Blocks(qm.Where(model.BlockColumns.Hash+"=?", jsonBlock.Hash)).OneG()
	if foundBlock != nil {
		block = foundBlock
	}

	block.Bits = jsonBlock.Bits
	block.Chainwork = jsonBlock.ChainWork
	block.Confirmations = uint(jsonBlock.Confirmations)
	block.Difficulty = jsonBlock.Difficulty
	block.Hash = jsonBlock.Hash
	block.Height = blockHeight
	block.MerkleRoot = jsonBlock.MerkleRoot
	block.NameClaimRoot = jsonBlock.NameClaimRoot
	block.Nonce = jsonBlock.Nonce
	block.PreviousBlockHash.SetValid(jsonBlock.PreviousBlockHash)
	block.NextBlockHash = null.NewString(jsonBlock.NextBlockHash, jsonBlock.NextBlockHash != "")
	block.BlockSize = uint64(jsonBlock.Size)
	block.BlockTime = uint64(jsonBlock.Time)
	block.Version = uint64(jsonBlock.Version)
	block.VersionHex = jsonBlock.VersionHex
	block.TXCount = int(jsonBlock.NTx)
	block.ProcessingState.SetValid(BlockProcessingStateProcessing)
	//block.TransactionHashes.SetValid(strings.Join(jsonBlock.Tx, ",")) //we don't need this, it's extremely redundant and heavy

	var err error
	if foundBlock != nil {
		err = block.UpdateG(boil.Infer())
	} else {
		err = block.InsertG(boil.Infer())
	}
	if err != nil {
		return nil, errors.Err(err)
	}

	return block, nil
}

func processGenesisBlock() error {
	genesisVerbose, genesis, err := lbrycrd.GetGenesisBlock()
	if err != nil {
		return errors.Err(err)
	}
	//This is an important lock to make sure we don't concurrently save transaction inputs/outputs accidentally via the
	// mempool sync.
	BlockLock.Lock()
	defer BlockLock.Unlock()
	block, err := parseBlockInfo(0, genesis)
	if err != nil {
		return errors.Err(err)
	}
	for _, tx := range genesisVerbose.Tx {
		tx.BlockHash = genesis.Hash
		err := ProcessTx(&tx, block.BlockTime, 0)
		if err != nil {
			return errors.Err(err)
		}
	}
	return errors.Err(markBlockProcessingState(block, BlockProcessingStateComplete))
}

func CleanupIncompleteHead() error {
	return errors.Err(cleanupIncompleteHead())
}

func cleanupIncompleteHead() error {
	BlockLock.Lock()
	defer BlockLock.Unlock()

	head, err := chainHeadBlock()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return errors.Err(err)
	}

	complete, err := blockHasExpectedTransactions(head)
	if err != nil {
		return errors.Err(err)
	}
	if !complete || head.ProcessingState.String == BlockProcessingStateProcessing || head.ProcessingState.String == BlockProcessingStateIncomplete {
		head.ProcessingState.SetValid(BlockProcessingStateIncomplete)
		err = markBlockProcessingState(head, BlockProcessingStateIncomplete)
		if err != nil {
			return errors.Err(err)
		}
		return errors.Err(deleteBlockWithRetry(head))
	}
	if !head.ProcessingState.Valid {
		return errors.Err(markBlockProcessingState(head, BlockProcessingStateComplete))
	}
	return nil
}

func MarkIncompleteBlockHeight(height uint64) error {
	BlockLock.Lock()
	defer BlockLock.Unlock()
	block, err := model.Blocks(
		model.BlockWhere.Height.EQ(height),
		model.BlockWhere.Hash.NEQ(MempoolBlockHash),
		qm.OrderBy(model.BlockColumns.ID+" DESC"),
		qm.Limit(1),
	).OneG()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return errors.Err(err)
	}
	return errors.Err(markBlockProcessingState(block, BlockProcessingStateIncomplete))
}

func backfillLegacyBlockStates(limit int) (int, error) {
	if limit <= 0 {
		limit = LegacyBlockBackfillBatchSize
	}
	head, err := chainHeadBlock()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, errors.Err(err)
	}
	blocks, err := model.Blocks(
		model.BlockWhere.ProcessingState.IsNull(),
		model.BlockWhere.Hash.NEQ(MempoolBlockHash),
		model.BlockWhere.ID.NEQ(head.ID),
		qm.OrderBy(model.BlockColumns.Height+" ASC"),
		qm.Limit(limit),
	).AllG()
	if err != nil {
		return 0, errors.Err(err)
	}
	for _, block := range blocks {
		complete, err := blockHasExpectedTransactions(block)
		if err != nil {
			return 0, errors.Err(err)
		}
		if !complete {
			return 0, errors.Base("legacy block %d (%s) failed transaction consistency validation", block.Height, block.Hash)
		}
		err = markBlockProcessingState(block, BlockProcessingStateComplete)
		if err != nil {
			return 0, errors.Err(err)
		}
	}
	return len(blocks), nil
}

func BackfillLegacyBlockStates(limit int) (int, error) {
	return backfillLegacyBlockStates(limit)
}

func chainHeadBlock() (*model.Block, error) {
	return model.Blocks(
		model.BlockWhere.Hash.NEQ(MempoolBlockHash),
		qm.OrderBy(model.BlockColumns.Height+" DESC"),
		qm.Limit(1),
	).OneG()
}

func blockHasExpectedTransactions(block *model.Block) (bool, error) {
	transactions, err := model.Transactions(model.TransactionWhere.BlockHashID.EQ(null.StringFrom(block.Hash))).AllG()
	if err != nil {
		return false, errors.Err(err)
	}
	if len(transactions) != block.TXCount {
		return false, nil
	}
	for _, transaction := range transactions {
		complete, err := transactionHasExpectedChildren(transaction)
		if err != nil {
			return false, errors.Err(err)
		}
		if !complete {
			return false, nil
		}
	}
	return true, nil
}

func transactionHasExpectedChildren(transaction *model.Transaction) (bool, error) {
	inputCount, err := model.Inputs(model.InputWhere.TransactionHash.EQ(transaction.Hash)).CountG()
	if err != nil {
		return false, errors.Err(err)
	}
	outputCount, err := model.Outputs(model.OutputWhere.TransactionHash.EQ(transaction.Hash)).CountG()
	if err != nil {
		return false, errors.Err(err)
	}
	return inputCount == int64(transaction.InputCount) && outputCount == int64(transaction.OutputCount), nil
}

func markBlockProcessingState(block *model.Block, state string) error {
	block.ProcessingState.SetValid(state)
	return errors.Err(block.UpdateG(boil.Whitelist(model.BlockColumns.ProcessingState)))
}

func deleteBlockWithRetry(block *model.Block) error {
	var err error
	for attempt := 0; attempt < blockDeleteRetryAttempts; attempt++ {
		err = block.DeleteG()
		if err == nil {
			return nil
		}
		time.Sleep(time.Duration(attempt+1) * blockDeleteRetryDelay)
	}
	return errors.Err(err)
}

type txSyncManager struct {
	daemonStopper *stop.Group
	queueStopper  *stop.Group
	syncStopper   *stop.Group
	workerStopper *stop.Group
	resultsCh     chan txProcessResult
	redoJobsCh    chan txToProcess
	jobsCh        chan txToProcess
	errorsCh      chan error
}

func syncTransactionsOfBlock(stopper *stop.Group, txs []string, blockTime uint64, blockHeight uint64) error {
	return syncTransactionsDependencyAware(stopper, txs, blockTime, blockHeight)
}

func syncTransactionsOfBlockLegacy(stopper *stop.Group, txs []string, blockTime uint64, blockHeight uint64) error {
	q("SYNC: started - " + strconv.Itoa(int(blockHeight)))
	// Initialization
	const maxErrorsPerBlockSync = 2
	// Error handling logic can be tested by decreasing the number of times a transaction can fail to 0.
	manager := txSyncManager{
		daemonStopper: stop.New(stopper),
		queueStopper:  stop.New(nil),
		syncStopper:   stop.New(nil),
		workerStopper: stop.New(nil),
		errorsCh:      make(chan error, maxErrorsPerBlockSync),
		resultsCh:     make(chan txProcessResult),
		redoJobsCh:    make(chan txToProcess, len(txs)),
		jobsCh:        make(chan txToProcess),
	}
	workers := util.Min(len(txs), MaxParallelTxProcessing)
	initTxWorkers(manager.workerStopper, workers, manager.jobsCh, manager.resultsCh)
	// Queue up n threads of transactions
	manager.queueStopper.Add(1)
	go queueTx(txs, blockTime, blockHeight, &manager)
	q("SYNC: launched queueing")
	//Setup reprocessing queue
	manager.syncStopper.Add(1)
	go reprocessQueue(&manager)
	// Handle the results
	manager.syncStopper.Add(1)
	go handleTxResults(len(txs), &manager)
	q("SYNC: launched handling")
	// Check for queueing errors ( ie. lbrycrd fetch)
	err, ok := waitForTxSyncError(&manager)
	if !ok {
		stopTxSyncManager(&manager, blockHeight)
		return ManualShutDownError
	}
	if err != nil {
		stopTxSyncManager(&manager, blockHeight)
		return errors.Err(err)
	}
	q("SYNC: received 1st on errorCh")

	// Wait for first handling error/nil first error is sent when tx fails x times or daemon is shutdown
	err, ok = waitForTxSyncError(&manager)
	if !ok {
		err = ManualShutDownError
	}
	q("SYNC: received 2nd on errorCh")
	stopTxSyncManager(&manager, blockHeight)
	return err
}

func waitForTxSyncError(manager *txSyncManager) (error, bool) {
	select {
	case err := <-manager.errorsCh:
		return err, true
	case <-manager.daemonStopper.Ch():
		return nil, false
	}
}

func stopTxSyncManager(manager *txSyncManager, blockHeight uint64) {
	q("SYNC: stop workers...")
	manager.queueStopper.Stop()
	manager.workerStopper.Stop()
	q("SYNC: stop sync...")
	manager.syncStopper.Stop()
	q("SYNC: wait for queue... - " + strconv.Itoa(int(blockHeight)))
	manager.queueStopper.StopAndWait()
	q("SYNC: wait for workers... - " + strconv.Itoa(int(blockHeight)))
	manager.workerStopper.StopAndWait()
	q("SYNC: wait for sync - " + strconv.Itoa(int(blockHeight)))
	manager.syncStopper.StopAndWait()
	q("SYNC: stopped - " + strconv.Itoa(int(blockHeight)))
	q("SYNC: closing redo channel")
	close(manager.redoJobsCh)
	q("SYNC: closing result channel")
	close(manager.resultsCh)
	q("SYNC: closing error channel")
	close(manager.errorsCh)
	q("SYNC: closing jobs channel")
	close(manager.jobsCh)
	q("SYNC: finished - " + strconv.Itoa(int(blockHeight)))
}

// q enables extensive logging on the concurrency of Chainquery. If there is every a deadlock and it's reproducible
// you can use this to debug it. Don't get stuck on the 'q' name either. It was literally a rare single letter that
// I chose, thats' it.
func q(a string) {
	if false {
		println(a)
	}
}

// MaxFailures tells Chainquery how many retryable failures a transaction can have before we roll back the block.
var MaxFailures int

func handleTxResults(nrToHandle int, manager *txSyncManager) {
	defer manager.syncStopper.Done()
	q("HANDLE: start handling")
	leftToProcess := nrToHandle
	for {
		q("HANDLE: waiting for next result...")
		select {
		case <-manager.daemonStopper.Ch():
			logrus.Info("stopping tx sync...")
			handleFailure(errors.Err("Daemon stopped manually!"), manager)
			return
		case <-manager.syncStopper.Ch():
			q("HANDLE: stopping handling...")
			return
		case txResult := <-manager.resultsCh:
			q("HANDLE: start handling new result.." + txResult.tx.Txid)
			leftToProcess--
			if txResult.failcount > MaxFailures {
				err := errors.Prefix("transaction "+txResult.tx.Txid+" failed more than "+strconv.Itoa(MaxFailures)+" times", txResult.err)
				handleFailure(err, manager)
				continue
			}
			if txResult.err != nil { // Try again if fails this time.
				leftToProcess++
				logrus.Warnf("retrying transaction %s for block %d after failure %d/%d: %s", txResult.tx.Txid, txResult.blockHeight, txResult.failcount+1, MaxFailures, txResult.err.Error())
				q("HANDLE: start sending to worker..." + txResult.tx.Txid)
				select {
				case manager.redoJobsCh <- txToProcess{tx: txResult.tx, blockTime: txResult.blockTime, failcount: txResult.failcount, blockHeight: txResult.blockHeight}:
				case <-manager.syncStopper.Ch():
					return
				case <-manager.daemonStopper.Ch():
					handleFailure(ManualShutDownError, manager)
					return
				}
				q("HANDLE: end sending to worker..." + txResult.tx.Txid)
				q("HANDLE: finish handling new result.." + txResult.tx.Txid)
				//continue
			}
			if leftToProcess == 0 {
				q("HANDLE: start passing done..")
				sendTxSyncError(manager, nil)
				q("HANDLE: end passing done..")
				q("HANDLE: end handling..")
				return
			}
			q("HANDLE: go to next loop..")
			continue
		}
	}
}

func queueTx(txs []string, blockTime uint64, blockHeight uint64, manager *txSyncManager) {
	defer manager.queueStopper.Done()
	q("QUEUE: start of queuing")
	txRawMap := make(map[string]*lbrycrd.TxRawResult)
	txSlice := make([]*lbrycrd.TxRawResult, len(txs))
	for i := range txs {
		select {
		case <-manager.queueStopper.Ch():
			q("QUEUE: stopping lbrycrd getting...")
			sendTxSyncError(manager, nil)
			return
		default:
			q("QUEUE:  start getting lbrycrd transaction..." + txs[i])
			jsonTx, err := lbrycrd.GetRawTransactionResponse(txs[i])
			if err != nil {
				sendTxSyncError(manager, errors.Prefix("GetRawTxError"+txs[i], err))
				return
			}
			txRawMap[jsonTx.Txid] = jsonTx
			txSlice[i] = jsonTx
			q("QUEUE:  end getting lbrycrd transaction..." + txs[i])
		}
	}
	txSet, ok := optimizeOrderToProcess(txRawMap)
	q("QUEUE: start interaction of " + strconv.Itoa(len(txSet)) + " transactions")
	if !ok {
		txSet = txSlice
	}
	for _, jsonTx := range txSet {
		select {
		case <-manager.queueStopper.Ch():
			q("QUEUE:  stopping processing " + jsonTx.Txid)
			sendTxSyncError(manager, nil)
			return
		case <-manager.daemonStopper.Ch():
			q("QUEUE: daemon stopping processing " + jsonTx.Txid)
			sendTxSyncError(manager, ManualShutDownError)
			return
		case manager.jobsCh <- txToProcess{tx: jsonTx, blockTime: blockTime, blockHeight: blockHeight}:
			q("QUEUE: start processing..." + jsonTx.Txid)
			q("QUEUE: end processing..." + jsonTx.Txid)
		}
	}
	q("QUEUE: end of queuing")
	sendTxSyncError(manager, nil)
	q("QUEUE: end of queuing...passed nil to errorCh")
}

// flush is a helper function for handling the results.
func flush(channel <-chan txProcessResult) {
	for range channel {
	}
}

func handleFailure(err error, manager *txSyncManager) {
	q("HANDLE: start passing error...")
	manager.queueStopper.Stop()
	q("HANDLE: flushing channel...")
	//Clears queue if any additional finished jobs come in at this point.
	// It also stops blocking during stopping.
	go flush(manager.resultsCh)
	q("HANDLE: stopping queue...")
	manager.queueStopper.StopAndWait()
	q("HANDLE: stopped queue...")
	q("HANDLE: stopping workers...")
	manager.workerStopper.Stop()
	q("HANDLE: stopped workers...")
	sendTxSyncError(manager, err)
	q("HANDLE: finish passing error...")
}

func sendTxSyncError(manager *txSyncManager, err error) bool {
	select {
	case manager.errorsCh <- err:
		return true
	case <-manager.daemonStopper.Ch():
		return false
	case <-manager.syncStopper.Ch():
		return false
	}
}

func getBlockToProcess(height *uint64) (*lbrycrd.GetBlockResponse, error) {
	hash, err := lbrycrd.GetBlockHash(*height)
	if err != nil {
		return nil, errors.Prefix(fmt.Sprintf("GetBlockHash Error(%d)", *height), err)
	}
	jsonBlock, err := lbrycrd.GetBlock(*hash)
	if err != nil {
		return nil, errors.Prefix("GetBlock Error("+*hash+")", err)
	}

	return jsonBlock, nil
}

var fetchBlockForReorg = getBlockToProcess

func checkHandleReorg(height uint64, chainPrevHash string) (uint64, error) {
	prevHeight := height - 1
	depth := 0
	if height > 0 {
		prevBlock, err := model.Blocks(qm.Where(model.BlockColumns.Height+"=?", prevHeight), qm.Load(model.BlockRels.BlockHashTransactions)).OneG()
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logrus.Warningf("missing previous block at height %d while processing %d; stepping back to fill the gap", prevHeight, height)
				return prevHeight, nil
			}
			return height, errors.Prefix("error getting block@"+strconv.Itoa(int(prevHeight)), err)
		}
		//Recursively delete blocks until they match or a reorg of depth 100 == failure of logic.
		for prevBlock.Hash != chainPrevHash && depth < 100 && prevHeight > 0 {
			hashes := make([]string, len(prevBlock.R.BlockHashTransactions))
			for i, th := range prevBlock.R.BlockHashTransactions {
				hashes[i] = th.Hash
			}
			fmt.Printf("block %s at height %d to be removed due to reorg. TX-> %s", prevBlock.Hash, prevBlock.Height, strings.Join(hashes, ","))
			logrus.Printf("block %s at height %d to be removed due to reorg. TX-> %s", prevBlock.Hash, prevBlock.Height, strings.Join(hashes, ","))
			// Delete because it needs to be reprocessed due to reorg
			err = prevBlock.DeleteG()
			if err != nil {
				return height, errors.Prefix("error deleting block@"+strconv.Itoa(int(prevHeight)), err)
			}

			depth++

			// Set chainPrevHash to new previous blocks prevhash to check next depth
			jsonBlock, err := fetchBlockForReorg(&prevHeight)
			if err != nil {
				return height, errors.Prefix("error getting block@"+strconv.Itoa(int(prevHeight))+" from lbrycrd", err)
			}
			chainPrevHash = jsonBlock.PreviousBlockHash

			// Decrement height and set prevBlock to the new previous
			prevHeight--
			prevBlock, err = model.Blocks(qm.Where(model.BlockColumns.Height+"=?", prevHeight), qm.Load(model.BlockRels.BlockHashTransactions)).OneG()
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					logrus.Warningf("missing previous block at height %d while handling reorg at %d; stepping back to fill the gap", prevHeight, height)
					return prevHeight, nil
				}
				return height, errors.Prefix("error getting previous block@"+strconv.Itoa(int(prevHeight)), err)
			}
		}
		if prevBlock.Hash != chainPrevHash {
			return height, errors.Base("reorg search exceeded limit at height %d without finding previous hash %s", height, chainPrevHash)
		}
		if depth > 0 {
			message := fmt.Sprintf("Reorg detected of depth %d at height %d,(last matching height %d) handling reorg processing!", depth, height, prevHeight)
			logrus.WithFields(logrus.Fields{
				"depth":                depth,
				"height":               height,
				"last_matching_height": prevHeight,
			}).Warning(message)
			return prevHeight, nil
		}
	}
	return height, nil
}

func reprocessQueue(manager *txSyncManager) {
	defer manager.syncStopper.Done()
	for {
		select {
		case <-manager.syncStopper.Ch():
			q("REDO: stopping redo jobs")
			return
		case redoJob, ok := <-manager.redoJobsCh:
			if !ok {
				return
			}
			q("REDO: start send new redo job - " + redoJob.tx.Txid)
			select {
			case manager.jobsCh <- redoJob:
			case <-manager.syncStopper.Ch():
				return
			}
			q("REDO: end send new redo job - " + redoJob.tx.Txid)
		}
	}
}
func optimizeOrderToProcess(txMap map[string]*lbrycrd.TxRawResult) ([]*lbrycrd.TxRawResult, bool) {
	txIDs := make([]string, 0, len(txMap))
	inDegree := make(map[string]int, len(txMap))
	children := make(map[string][]string, len(txMap))
	for txID := range txMap {
		txIDs = append(txIDs, txID)
		inDegree[txID] = 0
	}
	sort.Strings(txIDs)
	for _, txID := range txIDs {
		tx := txMap[txID]
		for _, vin := range tx.Vin {
			if _, ok := txMap[vin.TxID]; !ok {
				continue
			}
			children[vin.TxID] = append(children[vin.TxID], txID)
			inDegree[txID]++
		}
	}
	for parent := range children {
		sort.Strings(children[parent])
	}

	ready := make([]string, 0, len(txIDs))
	for _, txID := range txIDs {
		if inDegree[txID] == 0 {
			ready = append(ready, txID)
		}
	}
	orderedTx := make([]*lbrycrd.TxRawResult, 0, len(txMap))
	for len(ready) > 0 {
		txID := ready[0]
		ready = ready[1:]
		orderedTx = append(orderedTx, txMap[txID])
		q("Tx " + txID + " ordered")
		for _, childID := range children[txID] {
			inDegree[childID]--
			if inDegree[childID] == 0 {
				ready = append(ready, childID)
				sort.Strings(ready)
			}
		}
	}
	if len(orderedTx) != len(txMap) {
		orderedTx = orderedTx[:0]
		for _, txID := range txIDs {
			orderedTx = append(orderedTx, txMap[txID])
		}
		return orderedTx, false
	}
	return orderedTx, true
}
