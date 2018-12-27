package jobs

import (
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/errors"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
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
	transactionValue := transactionTbl + "." + model.TransactionColumns.Value
	transactionID := transactionTbl + "." + model.TransactionColumns.ID
	taCreditAmount := model.TransactionAddressColumns.CreditAmount
	taDebitAmount := model.TransactionAddressColumns.DebitAmount
	taTransactionID := model.TransactionAddressColumns.TransactionID

	result, err := boil.GetDB().Exec(`
		UPDATE ` + transactionTbl + `
		SET ` + transactionValue + ` = (
				SELECT COALESCE( SUM( ta.` + taCreditAmount + ` - ta.` + taDebitAmount + ` ),0.0) 
				FROM ` + transactionAddressTbl + ` ta 
				WHERE ta.` + taTransactionID + ` = ` + transactionID + `)`)
	if err != nil {
		return 0, errors.Prefix(syncTransactionValues, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Prefix(syncTransactionValues, err)
	}
	if rowsAffected > 0 {
		logrus.Warn(syncTransactionValues+" rows affected ( ", rowsAffected, " )")
	}

	return rowsAffected, nil

}
