package e2e

import (
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/lbry.go/extras/errors"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

func createBaseRawTx(inputs []btcjson.TransactionInput, change float64) (*wire.MsgTx, error) {
	addresses := make(map[btcutil.Address]btcutil.Amount)
	changeAddress, err := lbrycrd.LBRYcrdClient.GetNewAddress("")
	if err != nil {
		return nil, errors.Err(err)
	}
	changeAmount, err := btcutil.NewAmount(change)
	if err != nil {
		return nil, errors.Err(err)
	}
	addresses[changeAddress] = changeAmount
	lockTime := int64(0)
	return lbrycrd.LBRYcrdClient.CreateRawTransaction(inputs, addresses, &lockTime)
}

func getEmptyTx(totalOutputSpend float64) (*wire.MsgTx, error) {
	totalFees := 0.1
	unspentResults, err := lbrycrd.LBRYcrdClient.ListUnspentMin(1)
	if err != nil {
		return nil, errors.Err(err)
	}
	finder := newOutputFinder(unspentResults)

	outputs, err := finder.nextBatch(totalOutputSpend + totalFees)
	if err != nil {
		return nil, err
	}
	if len(outputs) == 0 {
		return nil, errors.Err("Not enough spendable outputs to create transaction")
	}
	inputs := make([]btcjson.TransactionInput, len(outputs))
	var totalInputSpend float64
	for i, output := range outputs {
		inputs[i] = btcjson.TransactionInput{Txid: output.TxID, Vout: output.Vout}
		totalInputSpend = totalInputSpend + output.Amount
	}

	change := totalInputSpend - totalOutputSpend - totalFees
	return createBaseRawTx(inputs, change)
}

func signTxAndSend(rawTx *wire.MsgTx) (*chainhash.Hash, error) {
	signedTx, allInputsSigned, err := lbrycrd.LBRYcrdClient.SignRawTransactionWithWallet(rawTx)
	if err != nil {
		return nil, errors.Err(err)
	}
	if !allInputsSigned {
		return nil, errors.Err("Not all inputs for the tx could be signed!")
	}

	return lbrycrd.LBRYcrdClient.SendRawTransaction(signedTx, false)
}
