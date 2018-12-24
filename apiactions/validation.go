package apiactions

import (
	"math"
	"net/http"

	"github.com/lbryio/chainquery/auth"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"

	"github.com/lbryio/lbry.go/api"
	"github.com/lbryio/lbry.go/errors"
	v "github.com/lbryio/ozzo-validation"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

var debug = false

func ValidateChainData(r *http.Request) api.Response {
	params := struct {
		From uint64
		To   *uint64
		Key  string
	}{}

	err := api.FormValues(r, &params, []*v.FieldRules{
		v.Field(&params.From, v.Required),
		v.Field(&params.To),
		v.Field(&params.Key),
	})
	if err != nil {
		return api.Response{Error: err, Status: http.StatusBadRequest}
	}

	if !auth.IsAuthorized(params.Key) {
		return api.Response{Error: errors.Err("not authorized"), Status: http.StatusUnauthorized}
	}

	if params.From < 0 || (params.To != nil && *params.To < 0) {
		return api.Response{Error: errors.Err("a positive from,to value must be passed"), Status: http.StatusBadRequest}
	}

	var missing []blockData
	if params.To != nil {
		missing, err = ValidateChain(&params.From, params.To)
	} else {
		missing, err = ValidateChain(&params.From, nil)
	}

	if err != nil {
		return api.Response{Error: err}
	}

	return api.Response{Data: missing}
}

type blockData struct {
	Block          uint64
	TxHash         string
	MissingOutputs float64
	MissingInputs  float64
}

func ValidateChain(from, to *uint64) ([]blockData, error) {

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

	var missingData = []blockData{}

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
						d("Validating Chain Data: tx %s missing %d outputs, %d inputs", lbryTxHash, math.Abs(float64(int(nrOutputs)-len(lbryTx.Vout))), math.Abs(float64(int(nrInputs)-len(lbryTx.Vin))))
						missingData = append(missingData,
							blockData{
								Block:          *from,
								TxHash:         lbryTxHash,
								MissingInputs:  math.Abs(float64(int(nrOutputs) - len(lbryTx.Vout))),
								MissingOutputs: math.Abs(float64(int(nrInputs) - len(lbryTx.Vin))),
							})
					}
				} else {
					d("Validating Chain Data: transaction %s missing", lbryTxHash)
					missingData = append(missingData, blockData{Block: *from, TxHash: lbryTxHash})
				}
			}
		} else {
			d("Validating Chain Data: block %d missing", *from)
			missingData = append(missingData, blockData{Block: *from})
		}
		*from++
		if *from%1000 == 0 {
			d("Validating Chain Data: at height %d", *from)
		}
	}

	return missingData, nil
}

func d(string string, args ...interface{}) {
	if debug {
		logrus.Warnf(string, args...)
	}
}
