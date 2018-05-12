package upgrademanager

import (
	"github.com/lbryio/chainquery/daemon/processing"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

func reProcessAllClaims() {
	outputs := model.OutputsG(qm.Where(model.OutputColumns.Type+" =?", lbrycrd.NonStandard),
		qm.Select(model.OutputColumns.TransactionHash)).AllP()
	for i, output := range outputs {
		processClaimOut(i, len(outputs), output.TransactionHash)
	}
}

func processClaimOut(index int, total int, txHash string) {
	tx, err := model.TransactionsG(qm.Where(model.TransactionColumns.Hash+"=?", txHash),
		qm.Select(model.TransactionColumns.Hash, model.TransactionColumns.BlockByHashID)).One()
	if err != nil {
		logrus.Panic(err)
	}
	txResult, err := lbrycrd.GetRawTransactionResponse(tx.Hash)
	if err != nil {
		logrus.Panic(err)
	}

	block, err := model.BlocksG(qm.Where(model.BlockColumns.Hash+"=?", txResult.BlockHash)).One()
	if err != nil {
		logrus.Panic(err)
	}
	if index%50 == 0 {
		logrus.Info("(", index, "/", total, ")", "Processing@Height ", block.Height)
	}

	err = processing.ProcessTx(txResult, block.BlockTime)
	if err != nil {
		logrus.Error("Reprocess Claim Error: ", err)
	}
}
