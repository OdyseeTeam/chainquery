package processing

import (
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/chainquery/twilio"
	"github.com/lbryio/chainquery/util"
	"github.com/lbryio/lbry.go/extras/errors"
	"github.com/lbryio/lbry.go/extras/stop"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

// BlockLock is used to lock block processing to a single parent thread.
var BlockLock = sync.Mutex{}

// ManualShutDownError Error with special handling. Used to stop the concurrency pipeline for processing blocks midway.
var ManualShutDownError = errors.Err("Daemon stopped manually!")

// RunBlockProcessing runs the processing of a block at a specific height. While any height can be passed in it is
// important to note that if the previous block is not processed it will panic to prevent corruption because blocks
// must be processed in order.
func RunBlockProcessing(stopper *stop.Group, height uint64) uint64 {
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
		rollBackHeight := height - 1
		blockRemovalError := block.DeleteG()
		if blockRemovalError != nil {
			logrus.Panicf("Could not delete block with bad data. Data corruption imminent at height %d. The block must be remove manually to continue.", height)
		}
		if err.Error() == ManualShutDownError.Error() {
			return rollBackHeight
		}
		logrus.Error("Block Processing Error: ", err)
		logrus.Warning("Ran into transaction sync error at height", height, ". Rolling block back to height", height-1)
		//ToDo - Should just return error...that is for another day
		return rollBackHeight
	}

	return height
}

// ProcessBlock processing a specific block and returns an error. Use this to process a block having a custom handling
// of the error.
func ProcessBlock(height uint64, stopper *stop.Group, jsonBlock *lbrycrd.GetBlockResponse) (*model.Block, error) {
	block := parseBlockInfo(height, jsonBlock)
	txs := jsonBlock.Tx
	return block, syncTransactionsOfBlock(stopper, txs, block.BlockTime, block.Height)
}

func parseBlockInfo(blockHeight uint64, jsonBlock *lbrycrd.GetBlockResponse) (block *model.Block) {

	block = &model.Block{}
	foundBlock, _ := model.Blocks(qm.Where(model.BlockColumns.Hash+"=?", jsonBlock.Hash)).OneG()
	if foundBlock != nil {
		block = foundBlock
	}

	block.Height = blockHeight
	block.Confirmations = uint(jsonBlock.Confirmations)
	block.Hash = jsonBlock.Hash
	block.BlockTime = uint64(jsonBlock.Time)
	block.Bits = jsonBlock.Bits
	block.BlockSize = uint64(jsonBlock.Size)
	block.Chainwork = jsonBlock.ChainWork
	difficulty, _ := strconv.ParseFloat(fmt.Sprintf("%.6f", jsonBlock.Difficulty), 64)
	block.Difficulty = difficulty
	block.MerkleRoot = jsonBlock.MerkleRoot
	block.NameClaimRoot = jsonBlock.NameClaimRoot
	block.Nonce = jsonBlock.Nonce
	block.NextBlockHash.String = jsonBlock.NextBlockHash
	block.PreviousBlockHash.SetValid(jsonBlock.PreviousBlockHash)
	block.TransactionHashes.SetValid(strings.Join(jsonBlock.Tx, ","))
	block.Version = uint64(jsonBlock.Version)
	block.VersionHex = jsonBlock.VersionHex

	var err error
	if foundBlock != nil {
		err = block.UpdateG(boil.Infer())
	} else {
		err = block.InsertG(boil.Infer())
	}
	if err != nil {
		logrus.Panic(err)
	}

	return block
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
	block := parseBlockInfo(0, genesis)
	for _, tx := range genesisVerbose.Tx {
		tx.BlockHash = genesis.Hash
		err := ProcessTx(&tx, block.BlockTime, 0)
		if err != nil {
			return errors.Err(err)
		}
	}
	return nil
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
		redoJobsCh:    make(chan txToProcess, 1000),
		jobsCh:        make(chan txToProcess),
	}
	workers := util.Min(len(txs), runtime.NumCPU())
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
	err := <-manager.errorsCh
	if err != nil {
		return errors.Err(err)
	}
	q("SYNC: received 1st on errorCh")

	// Wait for first handling error/nil first error is sent when tx fails x times or daemon is shutdown
	err = <-manager.errorsCh
	q("SYNC: received 2nd on errorCh")
	q("SYNC: stop workers...")
	manager.workerStopper.Stop()
	q("SYNC: stop sync...")
	manager.syncStopper.Stop()
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
	return err
}

//q enables extensive logging on the concurrency of Chainquery. If there is every a deadlock and it's reproducible
// you can use this to debug it. Don't get stuck on the 'q' name either. It was literally a rare single letter that
// I chose, thats' it.
func q(a string) {
	if false {
		println(a)
	}
}

// MaxFailures tells Chainquery how many failures a transaction can have before we rollback the block and try to process it
// it again. This is to stop an indefinite loop. Since transactions can be dependant on one another they can fail if not
// processed in the right order. We allow parallel processing by putting transactions into a queue, and if they fail to
// process, for example if its dependant transaction has not been processed yet, then we allow to go back into the queue
// x times ( MaxFailures ).
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
				err := errors.Prefix("transaction "+txResult.tx.Txid+" failed more than "+strconv.Itoa(MaxFailures)+" times!", txResult.err)
				handleFailure(err, manager)
				continue
			}
			if txResult.err != nil { // Try again if fails this time.
				leftToProcess++
				q("HANDLE: start sending to worker..." + txResult.tx.Txid)
				manager.redoJobsCh <- txToProcess{tx: txResult.tx, blockTime: txResult.blockTime, failcount: txResult.failcount, blockHeight: txResult.blockHeight}
				q("HANDLE: end sending to worker..." + txResult.tx.Txid)
				q("HANDLE: finish handling new result.." + txResult.tx.Txid)
				//continue
			}
			if leftToProcess == 0 {
				q("HANDLE: start passing done..")
				manager.errorsCh <- nil
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
	depthMap := make(map[string]int, len(txs))
	for i := range txs {
		select {
		case <-manager.queueStopper.Ch():
			q("QUEUE: stopping lbrycrd getting...")
			manager.errorsCh <- nil
			return
		default:
			q("QUEUE:  start getting lbrycrd transaction..." + txs[i])
			jsonTx, err := lbrycrd.GetRawTransactionResponse(txs[i])
			if err != nil {
				manager.errorsCh <- errors.Prefix("GetRawTxError:"+txs[i], err)
				return
			}
			txRawMap[jsonTx.Txid] = jsonTx
			depthMap[jsonTx.Txid] = 1
			q("QUEUE:  end getting lbrycrd transaction..." + txs[i])
		}
	}
	txSet := optimizeOrderToProcess(txRawMap, depthMap)
	q("QUEUE: start interation of " + strconv.Itoa(len(txSet)) + " transactions")
	for _, jsonTx := range txSet {
		select {
		case <-manager.queueStopper.Ch():
			q("QUEUE:  stopping processing " + jsonTx.Txid)
			manager.errorsCh <- nil
			return
		default:
			q("QUEUE: start processing..." + jsonTx.Txid)
			manager.jobsCh <- txToProcess{tx: jsonTx, blockTime: blockTime, blockHeight: blockHeight}
			q("QUEUE: end processing..." + jsonTx.Txid)
		}
	}
	q("QUEUE: end of queuing")
	manager.errorsCh <- nil
	q("QUEUE: end of queuing...passed nil to errorCh")
}

//flush is a helper function for handling the results.
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
	manager.errorsCh <- err
	q("HANDLE: finish passing error...")
}

func getBlockToProcess(height *uint64) (*lbrycrd.GetBlockResponse, error) {
	hash, err := lbrycrd.GetBlockHash(*height)
	if err != nil {
		return nil, errors.Prefix("GetBlockHash Error("+string(*height)+"): ", err)
	}
	jsonBlock, err := lbrycrd.GetBlock(*hash)
	if err != nil {
		return nil, errors.Prefix("GetBlock Error("+*hash+"): ", err)
	}

	return jsonBlock, nil
}

func checkHandleReorg(height uint64, chainPrevHash string) (uint64, error) {
	prevHeight := height - 1
	depth := 0
	if height > 0 {
		prevBlock, err := model.Blocks(qm.Where(model.BlockColumns.Height+"=?", prevHeight)).OneG()
		if err != nil {
			return height, errors.Prefix("error getting block@"+strconv.Itoa(int(prevHeight))+": ", err)
		}
		//Recursively delete blocks until they match or a reorg of depth 100 == failure of logic.
		for prevBlock.Hash != chainPrevHash && depth < 100 && prevHeight > 0 {
			// Delete because it needs to be reprocessed due to reorg
			logrus.Println("block ", prevBlock.Hash, " at height ", prevBlock.Height,
				" to be removed due to reorg. TX-> ", prevBlock.TransactionHashes)
			err = prevBlock.DeleteG()
			if err != nil {
				return height, errors.Prefix("error deleting block@"+strconv.Itoa(int(prevHeight))+": ", err)
			}

			depth++

			// Set chainPrevHash to new previous blocks prevhash to check next depth
			jsonBlock, err := getBlockToProcess(&prevHeight)
			if err != nil {
				return height, errors.Prefix("error getting block@"+strconv.Itoa(int(prevHeight))+" from lbrycrd: ", err)
			}
			chainPrevHash = jsonBlock.PreviousBlockHash

			// Decrement height and set prevBlock to the new previous
			prevHeight--
			prevBlock, err = model.Blocks(qm.Where(model.BlockColumns.Height+"=?", prevHeight)).OneG()
			if err != nil {
				return height, errors.Prefix("error getting previous block@"+strconv.Itoa(int(prevHeight))+": ", err)
			}
		}
		if depth > 0 {
			message := fmt.Sprintf("Reorg detected of depth %d at height %d,(last matching height %d) handling reorg processing!", depth, height, prevHeight)
			logrus.Warning(message)
			if depth > 2 {
				twilio.SendSMS(message)
			}
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
		case redoJob := <-manager.redoJobsCh:
			q("REDO: start send new redo job - " + redoJob.tx.Txid)
			manager.jobsCh <- redoJob
			q("REDO: end send new redo job - " + redoJob.tx.Txid)
		}
	}
}

func optimizeOrderToProcess(txMap map[string]*lbrycrd.TxRawResult, depthMap map[string]int) []*lbrycrd.TxRawResult {

	for _, tx := range txMap {
		for _, vin := range tx.Vin {
			if _, ok := txMap[vin.TxID]; ok {
				depthMap[vin.TxID] = (depthMap[vin.TxID] + 1) * 3
				depthMap[tx.Txid] = depthMap[tx.Txid] - 1
			}
		}
	}

	type depthPair struct {
		TxID  string
		Count int
	}

	var list []depthPair
	for k, v := range depthMap {
		list = append(list, depthPair{k, v})
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Count > list[j].Count
	})
	orderedTx := make([]*lbrycrd.TxRawResult, len(list))
	for i := range list {
		orderedTx[i] = txMap[list[i].TxID]
		//Additional debugging to output the order in which transactions are processed.
		q("Tx " + list[i].TxID + " Count " + strconv.Itoa(list[i].Count))
	}

	return orderedTx
}
