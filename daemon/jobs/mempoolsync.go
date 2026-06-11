package jobs

import (
	"database/sql"
	"sort"
	"sync/atomic"
	"time"

	"github.com/lbryio/chainquery/daemon/processing"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/metrics"
	"github.com/lbryio/chainquery/model"

	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

var mempoolSyncIsRunning atomic.Bool
var mempoolBlock *model.Block

// MempoolSync synchronizes the memory pool of lbrycrd. Transactions are processed against a special block with the
// Hash of the mempool constant. Transactions are processed recursively since transactions in the pool can be dependent
// on one another. The dependent transactions are always processed first.
func MempoolSync() {
	if !mempoolSyncIsRunning.CompareAndSwap(false, true) {
		return
	}
	resetRunning := true
	defer finishMempoolSync(&resetRunning)
	metrics.JobLoad.WithLabelValues("mempool_sync").Inc()
	defer metrics.JobLoad.WithLabelValues("mempool_sync").Dec()
	defer metrics.Job(time.Now(), "mempool_sync")

	logrus.Debug("Mempool Sync Started")
	txSet, err := lbrycrd.GetRawMempool()
	if err != nil {
		logrus.Error("MempoolSync:", errors.Err(err))
		return
	}
	rawTxs, err := fetchMempoolRawTransactions(txSet)
	if err != nil {
		logrus.Error("MempoolSync:", errors.Err(err))
		return
	}

	processing.BlockLock.Lock()
	defer processing.BlockLock.Unlock()
	if mempoolBlock == nil {
		mempoolBlock, err = getMempoolBlock()
		if err != nil {
			logrus.Error("MempoolSync:", err)
			return
		}
	}
	lastBlock, err := model.Blocks(
		model.BlockWhere.Hash.NEQ(processing.MempoolBlockHash),
		qm.OrderBy(model.BlockColumns.Height+" DESC"),
		qm.Limit(1),
	).OneG()
	if err != nil {
		logrus.Error("MempoolSync:", err)
	}
	staleTxs, err := model.Transactions(
		model.TransactionWhere.BlockHashID.EQ(null.StringFrom(processing.MempoolBlockHash)),
		model.TransactionWhere.CreatedAt.LTE(time.Now().Add(-1*time.Hour))).AllG()
	if err != nil {
		logrus.Error("MempoolSync:", err)
	}

	running, err := processTxSet(txSet, rawTxs, lastBlock, staleTxs)
	if err != nil {
		logrus.Debug("MempoolSync Error:", err)
	}
	if running {
		resetRunning = false
		go delayMempoolSyncReset()
	}
}

func finishMempoolSync(resetRunning *bool) {
	if *resetRunning {
		mempoolSyncIsRunning.Store(false)
	}
}

func delayMempoolSyncReset() {
	logrus.Info("Daemon is not caught up to mempool transactions, delaying mempool sync 1 minute...")
	time.Sleep(1 * time.Minute)
	mempoolSyncIsRunning.Store(false)
}

func getMempoolBlock() (*model.Block, error) {
	mempoolBlock, err := model.Blocks(model.BlockWhere.Hash.EQ(processing.MempoolBlockHash)).OneG()
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, errors.Err(err)
	}
	if mempoolBlock != nil {
		return mempoolBlock, nil
	}

	mempoolBlock = &model.Block{
		Height:        0,
		Confirmations: 0,
		Hash:          processing.MempoolBlockHash,
		BlockTime:     0,
		Bits:          "",
		BlockSize:     0,
		Chainwork:     "",
		Difficulty:    0,
		MerkleRoot:    "",
		NameClaimRoot: "",
		Nonce:         0,
		VersionHex:    "",
	}

	err = mempoolBlock.InsertG(boil.Infer())
	if err != nil {
		return nil, errors.Err(err)
	}

	return mempoolBlock, nil
}

func fetchMempoolRawTransactions(txSet lbrycrd.RawMempoolVerboseResponse) (map[string]*lbrycrd.TxRawResult, error) {
	txIDs := make(map[string]bool, len(txSet))
	for txID, txDetails := range txSet {
		txIDs[txID] = true
		for _, dependentTxID := range txDetails.Depends {
			txIDs[dependentTxID] = true
		}
	}
	orderedTxIDs := make([]string, 0, len(txIDs))
	for txID := range txIDs {
		orderedTxIDs = append(orderedTxIDs, txID)
	}
	sort.Strings(orderedTxIDs)
	rawTxs := make(map[string]*lbrycrd.TxRawResult, len(orderedTxIDs))
	for _, txID := range orderedTxIDs {
		txjson, err := lbrycrd.GetRawTransactionResponse(txID)
		if err != nil {
			return nil, errors.Err(err)
		}
		rawTxs[txID] = txjson
	}
	return rawTxs, nil
}

func processTxSet(txSet lbrycrd.RawMempoolVerboseResponse, rawTxs map[string]*lbrycrd.TxRawResult, lastBlock *model.Block, staleTxs model.TransactionSlice) (bool, error) {
	if lastBlock == nil {
		return false, errors.Base("cannot process mempool without a chain head block")
	}
	currTxMap := make(map[string]*model.Transaction)
	for _, tx := range staleTxs {
		currTxMap[tx.Hash] = tx
	}

	for txid, txDetails := range txSet {
		delete(currTxMap, txid)
		//Are we at the top of the chain?
		shouldProcessMempoolTransaction := lastBlock.Height+1 >= uint64(txDetails.Height)
		if shouldProcessMempoolTransaction {
			for _, dependentTxID := range txDetails.Depends {
				err := processMempoolTx(dependentTxID, *mempoolBlock, rawTxs[dependentTxID])
				if err != nil {
					return false, errors.Err(err)
				}
				delete(currTxMap, dependentTxID)
			}
			err := processMempoolTx(txid, *mempoolBlock, rawTxs[txid])
			if err != nil {
				return false, errors.Err(err)
			}
		} else {
			return true, nil
		}
	}
	for _, tx := range currTxMap {
		staleTx, err := model.Transactions(
			model.TransactionWhere.Hash.EQ(tx.Hash),
			model.TransactionWhere.BlockHashID.EQ(null.StringFrom(processing.MempoolBlockHash)),
		).OneG()
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return false, errors.Err(err)
		}
		err = staleTx.DeleteG()
		if err != nil {
			return false, errors.Err(err)
		}
	}

	return false, nil
}

func processMempoolTx(txid string, block model.Block, txjson *lbrycrd.TxRawResult) error {
	existingTx, err := model.Transactions(model.TransactionWhere.Hash.EQ(txid)).OneG()
	if err == nil {
		if existingTx.BlockHashID.Valid && existingTx.BlockHashID.String != processing.MempoolBlockHash {
			return nil
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return errors.Err(err)
	}
	if txjson == nil {
		return errors.Base("missing fetched mempool transaction %s", txid)
	}
	txjson.BlockHash = block.Hash
	return errors.Err(processing.ProcessTx(txjson, block.BlockTime, block.Height))
}
