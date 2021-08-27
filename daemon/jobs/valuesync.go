package jobs

import (
	"time"

	"github.com/lbryio/chainquery/metrics"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

const syncTransactionValues = "SyncTransactionValues: "
const syncAddressBalances = "SyncAddressBalances: "
const syncClaimsInChannel = "SyncClaimsInChannel: "

//SyncAddressBalancesJob runs the SyncAddressBalances as a background job.
func SyncAddressBalancesJob() {
	go func() {
		metrics.JobLoad.WithLabelValues("address_balance_sync").Inc()
		defer metrics.JobLoad.WithLabelValues("address_balance_sync").Dec()
		defer metrics.Job(time.Now(), "address_balance_sync")
		_, err := SyncAddressBalances()
		if err != nil {
			logrus.Error(syncAddressBalances, err)
		}
	}()
}

// SyncClaimsInChannelJob runs the SyncClaimsInChannel as a background job.
func SyncClaimsInChannelJob() {
	go func() {
		metrics.JobLoad.WithLabelValues("claims_in_channel_sync").Inc()
		defer metrics.JobLoad.WithLabelValues("claims_in_channel_sync").Dec()
		defer metrics.Job(time.Now(), "claims_in_channel_sync")
		err := SyncClaimCntInChannel()
		if err != nil {
			logrus.Error(syncClaimsInChannel, err)
		}
	}()
}

//TransactionValueSync synchronizes the transaction value column due to a bug in mysql related to triggers.
//https://bugs.mysql.com/bug.php?id=11472
func TransactionValueSync() {
	metrics.JobLoad.WithLabelValues("transaction_value_sync").Inc()
	defer metrics.JobLoad.WithLabelValues("transaction_value_sync").Dec()
	defer metrics.Job(time.Now(), "transaction_value_sync")
	_, err := SyncTransactionValue()
	if err != nil {
		logrus.Error(syncTransactionValues, err)
	}
}

//TransactionValueASync runs the SyncAddressBalances as a background job.
func TransactionValueASync() {
	go TransactionValueSync()
}

//SyncAddressBalances will update the balance for every address if needed based on the transaction address table and
// returns the number of rows changed. Due to mysql bug https://bugs.mysql.com/bug.php?id=11472
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
	if updateIncrement >= latestHeight {
		updateIncrement = latestHeight
	}
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

// SyncClaimCntInChannel will sync up the number of claims that are part of a particular channel. This can be used as a
// calculated column in the claim table to get this figure fast in a query.
func SyncClaimCntInChannel() error {

	t := model.TableNames
	c := model.ClaimColumns

	query := `SELECT COUNT(*) FROM ` + t.Claim + ` WHERE ` + t.Claim + `.` + c.PublisherID + ` = ?`

	var from int
	var to int

	latestBlock, err := model.Blocks(qm.Select(model.BlockColumns.Height), qm.OrderBy(model.BlockColumns.Height+" DESC")).OneG()
	if err != nil {
		return errors.Prefix(syncClaimsInChannel, err)
	}
	if latestBlock.Height == 0 {
		return errors.Prefix(syncClaimsInChannel, errors.Err("latest height = 0 "))
	}
	latestHeight := int(latestBlock.Height)
	updateIncrement := 5000
	if updateIncrement >= latestHeight {
		updateIncrement = latestHeight
	}
	for i := 0; i < latestHeight/updateIncrement; i++ {
		from = i * updateIncrement
		to = (i + 1) * updateIncrement
		if to > latestHeight {
			to = latestHeight
		}
		channelsToProcess, err := model.Claims(
			model.ClaimWhere.Height.GTE(uint(from)),
			model.ClaimWhere.Height.LTE(uint(to)),
			model.ClaimWhere.ClaimType.EQ(2)).AllG()
		if err != nil {
			return errors.Prefix(syncClaimsInChannel, err)
		}
		for _, channel := range channelsToProcess {
			result := boil.GetDB().QueryRow(query, channel.ClaimID)
			if err != nil {
				return errors.Prefix(syncClaimsInChannel, err)
			}
			var cnt null.Uint64
			err := result.Scan(&cnt)
			if err != nil {
				return errors.Prefix(syncClaimsInChannel, err)
			}
			if channel.ClaimCount == int64(cnt.Uint64) {
				continue
			}
			channel.ClaimCount = int64(cnt.Uint64)
			channel.ModifiedAt = time.Now()
			err = channel.UpdateG(boil.Whitelist(c.ClaimCount, c.ModifiedAt))
			if err != nil {
				return errors.Prefix(syncClaimsInChannel, err)
			}
		}
	}

	return nil

}
