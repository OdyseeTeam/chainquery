package jobs

import (
	"database/sql"
	"time"

	"github.com/lbryio/chainquery/daemon/processing"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"

	"github.com/lbryio/lbry.go/extras/errors"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

const (
	mempool = "MEMPOOL"
)

var mempoolSyncIsRunning = false
var mempoolBlock *model.Block

// MempoolSync synchronizes the memory pool of lbrycrd. Transactions are processed against a special block with the
// Hash of the mempool constant. Transactions are processed recursively since transactions in the pool can be dependent
// on one another. The dependent transactions are always processed first.
func MempoolSync() {
	if !mempoolSyncIsRunning {
		mempoolSyncIsRunning = true
		// Need to lock block processing to avoid race condition where we are saving a mempool transaction after a block
		// has already started processing transactions. The mempool transaction could overwrite the block transaction
		// incorrectly.
		processing.BlockLock.Lock()
		defer processing.BlockLock.Unlock()
		logrus.Debug("Mempool Sync Started")
		if mempoolBlock == nil {
			var err error
			mempoolBlock, err = getMempoolBlock()
			if err != nil {
				logrus.Error("MempoolSync:", err)
				return
			}
		}
		txSet, err := lbrycrd.GetRawMempool()
		if err != nil {
			logrus.Error("MempoolSync:", errors.Err(err))
			return
		}
		lastBlock, err := model.Blocks(qm.OrderBy(model.BlockColumns.Height+" DESC"), qm.Limit(1)).OneG()
		if err != nil {
			logrus.Error("MempoolSync:", err)
		}
		// Grabbing stale transactions to clean up the mempool state in chainquery ie invalidated double spends.
		staleTxs, err := model.Transactions(
			model.TransactionWhere.BlockHashID.EQ(null.StringFrom("MEMPOOL")),
			// We only want to get the old transactions sitting in mempool. Txs leave the mempool before they are sent as
			// a block. So we could end up deleting a tx, before we process it in a block, which for a claim update would
			// delete the original claim. There is still a change this could happen if a claim update tx sits in the
			// mempool for more than an hour.
			model.TransactionWhere.CreatedAt.LTE(time.Now().Add(-1*time.Hour))).AllG()
		if err != nil {
			logrus.Error("MempoolSync:", err)
		}

		running, err := processTxSet(txSet, lastBlock, staleTxs)
		if err != nil {
			logrus.Error("MempoolSync:", err)
		}

		mempoolSyncIsRunning = running
	}
}

func getMempoolBlock() (*model.Block, error) {
	mempoolBlock, err := model.Blocks(qm.Where(model.BlockColumns.Hash+" = ?", "MEMPOOL")).OneG()
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, errors.Err(err)
	}
	if mempoolBlock != nil {
		return mempoolBlock, nil
	}

	mempoolBlock = &model.Block{
		Height:        0,
		Confirmations: 0,
		Hash:          mempool,
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

func processTxSet(txSet lbrycrd.RawMempoolVerboseResponse, lastBlock *model.Block, staleTxs model.TransactionSlice) (bool, error) {
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
				err := processMempoolTx(dependentTxID, *mempoolBlock)
				if err != nil {
					return false, errors.Err(err)
				}
				delete(currTxMap, dependentTxID)
			}
			err := processMempoolTx(txid, *mempoolBlock)
			if err != nil {
				return false, errors.Err(err)
			}
		} else {
			go func() {
				logrus.Info("Daemon is not caught up to mempool transactions, delaying mempool sync 1 minute...")
				time.Sleep(1 * time.Minute)
				mempoolSyncIsRunning = false
			}()
			return true, nil
		}
	}
	for _, tx := range currTxMap {
		err := tx.DeleteG()
		if err != nil {
			return false, errors.Err(err)
		}
	}

	return false, nil
}

func processMempoolTx(txid string, block model.Block) error {
	exists, err := model.Transactions(qm.Where(model.TransactionColumns.Hash+"=?", txid)).ExistsG()
	if err != nil {
		return errors.Err(err)
	}
	if !exists {
		txjson, err := lbrycrd.GetRawTransactionResponse(txid)
		if err != nil {
			return errors.Err(err)
		}
		txjson.BlockHash = block.Hash
		return errors.Err(processing.ProcessTx(txjson, block.BlockTime, block.Height))
	}
	return nil
}
