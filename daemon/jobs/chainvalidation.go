package jobs

import (
	"fmt"
	"time"

	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"

	"github.com/lbryio/lbry.go/extras/errors"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

var validatingChain = false
var debug = false

const chainValidationJob = "chainvalidationjob"

// ValidateChain goes through the entire chain to make sure the data matches what is in the block chain. If there are
// differences it will log an error message identifying the magnitude of the difference.
func ValidateChain() {
	if !validatingChain {
		go func() {
			var job *model.JobStatus
			exists, err := model.JobStatuses(qm.Where(model.JobStatusColumns.JobName+"=?", chainValidationJob)).ExistsG()
			if err != nil {
				logrus.Error("Chain Validation: ", err)
				return
			}
			if !exists {
				job = &model.JobStatus{JobName: chainValidationJob}
			} else {
				job, err = model.JobStatuses(qm.Where(model.JobStatusColumns.JobName+"=?", chainValidationJob)).OneG()
				if err != nil {
					logrus.Error("Chain Validation: ", err)
					return
				}
			}
			startOfChain := uint64(0)
			missingData, err := ValidateChainRange(&startOfChain, nil)
			if err != nil {
				job.ErrorMessage.SetValid(err.Error())
				job.IsSuccess = false
			}

			if len(missingData) > 0 {
				job.ErrorMessage.SetValid(fmt.Sprintf("%d pieces of missing data", len(missingData)))
			}

			job.LastSync = time.Now()

			err = job.UpsertG(boil.Infer(), boil.Infer())
			if err != nil {
				logrus.Error("Chain Validation: ", err)
				return
			}
		}()
	}
}

// BlockData type holds information about where differences are in Chainquery vs the Blockchain.
type BlockData struct {
	Block          uint64
	TxHash         string
	MissingOutputs int
	MissingInputs  int
}

// ValidateChainRange validates a range of blocks and returns the differences.
func ValidateChainRange(from, to *uint64) ([]BlockData, error) {

	if from == nil {
		start := uint64(0)
		from = &start
	}
	if to == nil {
		currHeight, err := lbrycrd.GetBlockCount()
		if err != nil {
			return nil, errors.Err(err)
		}
		to = currHeight
	}

	var missingData = make([]BlockData, 0)

	for *from <= *to {

		haveBlock, err := model.Blocks(qm.Select(model.BlockColumns.Hash), qm.Where(model.BlockColumns.Height+"=?", *from)).ExistsG()
		if err != nil {
			return nil, errors.Err(err)
		}
		block, err := model.Blocks(qm.Select(model.BlockColumns.Hash), qm.Where(model.BlockColumns.Height+"=?", *from)).OneG()
		if err != nil {
			return nil, errors.Err(err)
		}

		if haveBlock {
			hash, err := lbrycrd.GetBlockHash(*from)
			if err != nil {
				return nil, errors.Err(err)
			}

			lbryBlock, err := lbrycrd.GetBlock(*hash)
			if err != nil {
				return nil, errors.Err(err)
			}

			transactions, err := block.BlockHashTransactions(qm.Select(model.TransactionColumns.Hash, model.TransactionColumns.ID)).AllG()
			if err != nil {
				return nil, errors.Err(err)
			}
			missingData, err = checkTxs(missingData, lbryBlock, transactions)
			if err != nil {
				return nil, err
			}
		} else {
			d("Validating Chain Data: block %d missing", *from)
			missingData = append(missingData, BlockData{Block: *from})
		}
		*from++
		if *from%1000 == 0 {
			d("Validating Chain Data: at height %d", *from)
		}
	}

	return missingData, nil
}

func checkTxs(missingData []BlockData, lbryBlock *lbrycrd.GetBlockResponse, transactions model.TransactionSlice) ([]BlockData, error) {
	for _, lbryTxHash := range lbryBlock.Tx {
		var tx *model.Transaction
		for _, transaction := range transactions {
			if transaction.Hash == lbryTxHash {
				tx = transaction
			}
		}
		if tx != nil {
			lbryTx, err := lbrycrd.GetRawTransactionResponse(lbryTxHash)
			if err != nil {
				return nil, errors.Err(err)
			}
			nrOutputs, err := tx.Outputs().CountG()
			if err != nil {
				return nil, errors.Err(err)
			}
			nrInputs, err := tx.Inputs().CountG()
			if err != nil {
				return nil, errors.Err(err)
			}
			if int(nrOutputs) != len(lbryTx.Vout) || int(nrInputs) != len(lbryTx.Vin) {
				d("Validating Chain Data: tx %s missing %d outputs, %d inputs", lbryTxHash, len(lbryTx.Vout)-int(nrOutputs), len(lbryTx.Vin)-int(nrInputs))
				missingData = append(missingData,
					BlockData{
						Block:          uint64(lbryBlock.Height),
						TxHash:         lbryTxHash,
						MissingInputs:  len(lbryTx.Vin) - int(nrInputs),
						MissingOutputs: len(lbryTx.Vout) - int(nrOutputs),
					})
			}
		} else {
			d("Validating Chain Data: transaction %s missing", lbryTxHash)
			missingData = append(missingData, BlockData{Block: uint64(lbryBlock.Height), TxHash: lbryTxHash})
		}
	}
	return missingData, nil
}

func d(string string, args ...interface{}) {
	if debug {
		logrus.Warnf(string, args...)
	}
}
