package processing

import (
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/chainquery/util"
	"github.com/lbryio/lbry.go/errors"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

// RunBlockProcessing runs the processing of a block at a specific height. While any height can be passed in it is
// important to note that if the previous block is not processed it will panic to prevent corruption because blocks
// must be processed in order.
func RunBlockProcessing(height *uint64) {
	defer util.TimeTrack(time.Now(), "runBlockProcessing", "daemonprofile")
	jsonBlock, err := getBlockToProcess(height)
	if err != nil {
		logrus.Error("Get Block Error: ", err)
		return
	}
	block := &model.Block{}
	foundBlock, _ := model.BlocksG(qm.Where(model.BlockColumns.Hash+"=?", jsonBlock.Hash)).One()
	if foundBlock != nil {
		block = foundBlock
	}
	block.Height = *height
	block.Confirmations = uint(jsonBlock.Confirmations)
	block.Hash = jsonBlock.Hash
	block.BlockTime = uint64(jsonBlock.Time)
	block.Bits = jsonBlock.Bits
	block.BlockSize = uint64(jsonBlock.Size)
	block.Chainwork = jsonBlock.ChainWork
	block.Difficulty = jsonBlock.Difficulty
	block.MerkleRoot = jsonBlock.MerkleRoot
	block.NameClaimRoot = jsonBlock.NameClaimRoot
	block.NextBlockHash.String = jsonBlock.NextHash
	block.PreviousBlockHash.String = jsonBlock.PreviousHash
	block.TransactionHashes.String = strings.Join(jsonBlock.Tx, ",")
	block.Version = uint64(jsonBlock.Version)
	block.VersionHex = jsonBlock.VersionHex
	if foundBlock != nil {
		err = block.UpdateG()
	} else {
		err = block.InsertG()
	}
	if err != nil {
		logrus.Error(err)
	}

	txs := jsonBlock.Tx
	syncTransactionsOfBlock(txs, block.BlockTime)
}

func syncTransactionsOfBlock(txs []string, blockTime uint64) {
	txJobs := make(chan txToProcess, 100)
	errorchan := make(chan txProcessError, 100)
	workers := util.Min(len(txs), runtime.NumCPU())
	initTxWorkers(workers, txJobs, errorchan)

	// Queue up all transactions
	for i := range txs {
		go func(index int) {
			jsonTx, err := lbrycrd.GetRawTransactionResponse(txs[index])
			if err != nil {
				logrus.Error("GetRawTxError:", err)
			} else {
				go func() { txJobs <- txToProcess{tx: jsonTx, blockTime: blockTime} }()
			}
		}(i)
	}
	// Check for errors. If there is an error put it to the back of the queue.
	wg := sync.WaitGroup{}
	errorCheckCount := len(txs)
	wg.Add(1)
	go func(cnt int) {
		defer wg.Done()
		for i := 0; i < cnt; i++ {
			txError := <-errorchan
			if txError.failcount > 1000 {
				logrus.Panic(errors.Base("transaction " + txError.tx.Txid + " failed more than 1000 times!"))
			}
			if txError.err != nil {
				go func() {
					txJobs <- txToProcess{tx: txError.tx, blockTime: txError.blockTime, failcount: txError.failcount}
				}()
				cnt++
			}
		}
	}(errorCheckCount)

	wg.Wait()
	close(txJobs)
	close(errorchan)
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
