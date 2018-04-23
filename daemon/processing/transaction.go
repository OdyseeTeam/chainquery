package processing

import (
	"sync"
	"time"

	"github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/chainquery/util"
	"github.com/lbryio/lbry.go/errors"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"runtime"
)

type txDebitCredits struct {
	AddrDCMap map[string]*AddrDebitCredits
	mutex     *sync.RWMutex
}

func NewTxDebitCredits() *txDebitCredits {
	t := txDebitCredits{}
	v := make(map[string]*AddrDebitCredits)
	t.AddrDCMap = v
	t.mutex = &sync.RWMutex{}

	return &t

}

type AddrDebitCredits struct {
	debits  float64
	credits float64
}

func (addDC *AddrDebitCredits) Debits() float64 {
	return addDC.debits
}

func (addDC *AddrDebitCredits) Credits() float64 {
	return addDC.credits
}

func (txDC *txDebitCredits) subtract(address string, value float64) error {
	txDC.mutex.Lock()
	if txDC.AddrDCMap[address] == nil {
		addrDC := AddrDebitCredits{}
		txDC.AddrDCMap[address] = &addrDC
	}
	txDC.AddrDCMap[address].debits = txDC.AddrDCMap[address].debits + value
	txDC.mutex.Unlock()
	return nil
}

func (txDC *txDebitCredits) add(address string, value float64) error {
	txDC.mutex.Lock()
	if txDC.AddrDCMap[address] == nil {
		addrDC := AddrDebitCredits{}
		txDC.AddrDCMap[address] = &addrDC
	}
	txDC.AddrDCMap[address].credits = txDC.AddrDCMap[address].credits + value
	txDC.mutex.Unlock()

	return nil
}

func ProcessTx(jsonTx *lbrycrd.TxRawResult, blockTime uint64) error {
	defer util.TimeTrack(time.Now(), "processTx "+jsonTx.Txid+" -- ", "daemonprofile")
	transaction := &model.Transaction{}
	foundTx, err := model.TransactionsG(qm.Where(model.TransactionColumns.Hash+"=?", jsonTx.Txid)).One()
	if foundTx != nil {
		transaction = foundTx
	}
	transaction.Hash = jsonTx.Txid
	transaction.Version = int(jsonTx.Version)
	transaction.BlockByHashID.String = jsonTx.BlockHash
	transaction.BlockByHashID.Valid = true
	transaction.CreatedTime = time.Unix(0, jsonTx.Blocktime)
	transaction.TransactionTime.Uint64 = uint64(jsonTx.Blocktime)
	transaction.TransactionTime.Valid = true
	transaction.LockTime = uint(jsonTx.LockTime)
	transaction.InputCount = uint(len(jsonTx.Vin))
	transaction.OutputCount = uint(len(jsonTx.Vout))
	transaction.Raw.String = jsonTx.Hex
	transaction.TransactionSize = uint64(jsonTx.Size)
	transaction.Value = 0.0 //p.GetTotalValue(jsonTx.Vout)

	txDbCrAddrMap := NewTxDebitCredits()

	if foundTx != nil {
		transaction.Update(boil.GetDB())
	} else {
		err = transaction.Insert(boil.GetDB())
	}
	if err != nil {
		return err
	}
	//Save transaction before the id is used.
	_, err = createUpdateVoutAddresses(transaction, &jsonTx.Vout, blockTime)
	if err != nil {
		err := errors.Prefix("Vout Address Creation Error: ", err)
		return err
	}
	_, err = createUpdateVinAddresses(*transaction, &jsonTx.Vin, blockTime)
	if err != nil {
		err := errors.Prefix("Vin Address Creation Error: ", err)
		return err
	}

	vins := jsonTx.Vin
	vinjobs := make(chan vinToProcess, len(vins))
	errorchan := make(chan error, len(vins))
	workers := util.Min(len(vins), runtime.NumCPU())
	initVinWorkers(workers, vinjobs, errorchan)
	for i := range vins {
		index := i
		vinjobs <- vinToProcess{jsonVin: &vins[index], tx: transaction, txDC: txDbCrAddrMap}
	}
	close(vinjobs)
	for i := 0; i < len(vins); i++ {
		err := <-errorchan
		if err != nil {
			logrus.Error("Vin Error->", err)
			panic(err)
		}
	}
	close(errorchan)
	vouts := jsonTx.Vout
	voutjobs := make(chan voutToProcess, len(vouts))
	errorchan = make(chan error, len(vouts))
	workers = util.Min(len(vouts), runtime.NumCPU())
	initVoutWorkers(workers, voutjobs, errorchan)
	for i := range vouts {
		index := i
		voutjobs <- voutToProcess{jsonVout: &vouts[index], tx: transaction, txDC: txDbCrAddrMap}
	}
	close(voutjobs)
	for i := 0; i < len(vouts); i++ {
		err := <-errorchan
		if err != nil {
			logrus.Error("Vout Error->", err)
			panic(err)
		}
	}
	close(errorchan)
	for addr, DC := range txDbCrAddrMap.AddrDCMap {

		address := datastore.GetAddress(addr)

		txAddr := datastore.GetTxAddress(transaction.ID, address.ID)

		txAddr.CreditAmount = DC.Credits()
		txAddr.DebitAmount = DC.Debits()

		datastore.PutTxAddress(txAddr)

	}

	return err
}
