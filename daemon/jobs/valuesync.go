package jobs

import (
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/extras/errors"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

const syncTransactionValues = "SyncTransactionValues: "
const syncAddressBalances = "SyncAddressBalances: "

//SyncAddressBalancesJob runs the SyncAddressBalances as a background job.
func SyncAddressBalancesJob() {
	go func() {
		_, err := SyncAddressBalances()
		if err != nil {
			logrus.Error(syncAddressBalances, err)
		}
	}()
}

//SyncTransactionValueJob runs the SyncAddressBalances as a background job.
func SyncTransactionValueJob() {
	go func() {
		_, err := SyncTransactionValue()
		if err != nil {
			logrus.Error(syncTransactionValues, err)
		}
	}()
}

//SyncAddressBalances will update the balance for every address if needed based on the transaction address table and
// returns the number of rows changed.
func SyncAddressBalances() (int64, error) {

	addressTbl := model.TableNames.Address
	transactionAddressTbl := model.TableNames.TransactionAddress
	addressBalance := addressTbl + "." + model.AddressColumns.Balance
	addressID := addressTbl + "." + model.AddressColumns.ID
	taCreditAmount := model.TransactionAddressColumns.CreditAmount
	taDebitAmount := model.TransactionAddressColumns.DebitAmount
	taAddressID := model.TransactionAddressColumns.AddressID
	result, err := boil.GetDB().Exec(`
		UPDATE ` + addressTbl + `
		SET ` + addressBalance + ` = (
				SELECT COALESCE( SUM( ta.` + taCreditAmount + ` - ta.` + taDebitAmount + ` ),0.0) 
				FROM ` + transactionAddressTbl + ` ta 
				WHERE ta.` + taAddressID + ` = ` + addressID + `)`)
	if err != nil {
		return 0, errors.Prefix(syncAddressBalances, err)
	}

	if result == nil {
		println("result is nil.")
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Prefix(syncAddressBalances, err)
	}
	if rowsAffected > 0 {
		logrus.Warn(syncAddressBalances+" rows affected ( ", rowsAffected, " )")
	}

	return rowsAffected, nil

}

//SyncTransactionValue will sync up the value column of all transactions based on the transaction address table and
// returns the number of rows affected.
func SyncTransactionValue() (int64, error) {

	transactionTbl := model.TableNames.Transaction
	transactionAddressTbl := model.TableNames.TransactionAddress
	blockTbl := model.TableNames.Block
	transactionValue := transactionTbl + "." + model.TransactionColumns.Value
	transactionBlockHashID := transactionTbl + "." + model.TransactionColumns.BlockHashID
	transactionID := transactionTbl + "." + model.TransactionColumns.ID
	taCreditAmount := model.TransactionAddressColumns.CreditAmount
	taTransactionID := model.TransactionAddressColumns.TransactionID
	blockHash := blockTbl + "." + model.BlockColumns.Hash
	blockHeight := blockTbl + "." + model.BlockColumns.Height

	query := `
		UPDATE ` + transactionTbl + ` 
		INNER JOIN ` + blockTbl + ` ON ` + blockHash + ` = ` + transactionBlockHashID + `
		SET ` + transactionValue + ` =  (
			SELECT COALESCE( SUM( ta.` + taCreditAmount + ` ),0.0) 
			FROM ` + transactionAddressTbl + ` ta
			WHERE ta.` + taTransactionID + ` = ` + transactionID + ` )
		WHERE ` + blockHeight + ` BETWEEN ? AND ?`

	var from int
	var to int
	var affected int64

	latestBlock, err := model.Blocks(qm.Select(model.BlockColumns.Height), qm.OrderBy(model.BlockColumns.Height+" DESC")).OneG()
	if err != nil {
		return 0, errors.Prefix(syncTransactionValues, err)
	}
	if latestBlock.Height == 0 {
		return 0, errors.Prefix(syncTransactionValues, errors.Err("latest height = 0 "))
	}
	latestHeight := int(latestBlock.Height)
	updateIncrement := 5000
	for i := 0; i < latestHeight/updateIncrement; i++ {
		from = i * updateIncrement
		to = (i + 1) * updateIncrement
		if to > latestHeight {
			to = latestHeight
		}
		result, err := boil.GetDB().Exec(query, from, to)
		if err != nil {
			return 0, errors.Prefix(syncTransactionValues, err)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return 0, errors.Prefix(syncTransactionValues, err)
		}
		affected = affected + rowsAffected
	}

	if affected > 0 {
		logrus.Warn(syncTransactionValues+" rows affected ( ", affected, " )")
	}

	return affected, nil

}
