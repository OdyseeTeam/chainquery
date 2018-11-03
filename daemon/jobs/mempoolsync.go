package jobs

import (
	"database/sql"
	"github.com/lbryio/chainquery/daemon/processing"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/errors"
	"github.com/sirupsen/logrus"
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
		logrus.Debug("Mempool Sync Started")
		if mempoolBlock == nil {
			mempoolBlock = getMempoolBlock()
		}
		txSet, err := lbrycrd.GetRawMempool()
		if err != nil {
			logrus.Error("MempoolSync:", errors.Err(err))
			return
		}
		// Need to lock block processing to avoid race condition where we are saving a mempool transaction after a block
		// has already started processing transactions. The mempool transaction could overwrite the block transaction
		// incorrectly.
		processing.BlockLock.Lock()
		defer processing.BlockLock.Unlock()

		for txid, txDetails := range txSet {

			for _, dependentTxID := range txDetails.Depends {
				err := processMempoolTx(dependentTxID, *mempoolBlock)
				if err != nil {
					logrus.Error("MempoolSync:", errors.Err(err))
				}
			}
			err := processMempoolTx(txid, *mempoolBlock)
			if err != nil {
				logrus.Error("MempoolSync:", errors.Err(err))
			}
		}
		mempoolSyncIsRunning = false
	}
}

func getMempoolBlock() *model.Block {
	mempoolBlock, err := model.BlocksG(qm.Where(model.BlockColumns.Hash+" = ?", "MEMPOOL")).One()
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logrus.Error("Mempool:", err)
	}
	if mempoolBlock != nil {
		return mempoolBlock
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

	err = mempoolBlock.InsertG()
	if err != nil {
		logrus.Error("Mempool:", errors.Err(err))
	}

	return mempoolBlock
}

func processMempoolTx(txid string, block model.Block) error {
	txjson, err := lbrycrd.GetRawTransactionResponse(txid)
	if err != nil {
		return errors.Prefix("Mempool:", errors.Err(err))
	}
	txjson.BlockHash = block.Hash
	return errors.Err(processing.ProcessTx(txjson, block.BlockTime, block.Height))
}
