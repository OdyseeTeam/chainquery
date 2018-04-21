package upgrademanager

import (
	"github.com/lbryio/chainquery/daemon/processing"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

func ReProcessAllClaims() {
	outputs := model.OutputsG(qm.Where(model.OutputColumns.Type+" =?", lbrycrd.NON_STANDARD),
		qm.Select(model.OutputColumns.TransactionHash)).AllP()
	for _, output := range outputs {
		tx, err := model.TransactionsG(qm.Where(model.TransactionColumns.Hash+"=?", output.TransactionHash),
			qm.Select(model.TransactionColumns.Hash, model.TransactionColumns.BlockByHashID)).One()
		if err != nil {
			panic(err)
		}
		txResult, err := lbrycrd.GetRawTransactionResponse(tx.Hash)
		if err != nil {
			panic(err)
		}

		block, err := model.BlocksG(qm.Where(model.BlockColumns.Hash+"=?", txResult.BlockHash)).One()
		if err != nil {
			panic(err)
		}
		logrus.Debug("Processing ", block.Height, " ", tx.Hash)
		processing.ProcessTx(txResult, block.BlockTime)
	}
}
