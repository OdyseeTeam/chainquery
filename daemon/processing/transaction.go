package processing

import (
	"runtime"
	"sync"
	"time"

	"github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/chainquery/util"
	"github.com/lbryio/lbry.go/errors"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

type txDebitCredits struct {
	addrDCMap map[string]*addrDebitCredits
	mutex     *sync.RWMutex
}

func newTxDebitCredits() *txDebitCredits {
	t := txDebitCredits{}
	v := make(map[string]*addrDebitCredits)
	t.addrDCMap = v
	t.mutex = &sync.RWMutex{}

	return &t

}

type addrDebitCredits struct {
	debits  float64
	credits float64
}

func (addDC *addrDebitCredits) Debits() float64 {
	return addDC.debits
}

func (addDC *addrDebitCredits) Credits() float64 {
	return addDC.credits
}

func (txDC *txDebitCredits) subtract(address string, value float64) {
	txDC.mutex.Lock()
	if txDC.addrDCMap[address] == nil {
		addrDC := addrDebitCredits{}
		txDC.addrDCMap[address] = &addrDC
	}
	txDC.addrDCMap[address].debits = txDC.addrDCMap[address].debits + value
	txDC.mutex.Unlock()
}

func (txDC *txDebitCredits) add(address string, value float64) {
	txDC.mutex.Lock()
	if txDC.addrDCMap[address] == nil {
		addrDC := addrDebitCredits{}
		txDC.addrDCMap[address] = &addrDC
	}
	txDC.addrDCMap[address].credits = txDC.addrDCMap[address].credits + value
	txDC.mutex.Unlock()
}

// ProcessTx processes an individual transaction from a block.
func ProcessTx(jsonTx *lbrycrd.TxRawResult, blockTime uint64) error {
	defer util.TimeTrack(time.Now(), "processTx "+jsonTx.Txid+" -- ", "daemonprofile")

	//Save transaction before the id is used any where else otherwise it will be 0
	transaction, err := saveUpdateTransaction(jsonTx)
	if err != nil {
		return err
	}

	txDbCrAddrMap := newTxDebitCredits()

	_, err = createUpdateVoutAddresses(transaction, &jsonTx.Vout, blockTime)
	if err != nil {
		err := errors.Prefix("Vout Address Creation Error: ", err)
		return err
	}
	_, err = createUpdateVinAddresses(transaction, &jsonTx.Vin, blockTime)
	if err != nil {
		err := errors.Prefix("Vin Address Creation Error: ", err)
		return err
	}

	// Process the inputs of the tranasction
	saveUpdateInputs(transaction, jsonTx, txDbCrAddrMap)

	// Process the outputs of the transaction
	saveUpdateOutputs(transaction, jsonTx, txDbCrAddrMap)

	//Set the send and receive values for the transaction
	setSendReceive(transaction, txDbCrAddrMap)

	return nil
}

func saveUpdateTransaction(jsonTx *lbrycrd.TxRawResult) (*model.Transaction, error) {
	transaction := &model.Transaction{}
	foundTx, err := model.TransactionsG(qm.Where(model.TransactionColumns.Hash+"=?", jsonTx.Txid)).One()
	if err != nil {
		logrus.Error("Find Transaction Error: ", err)
	}
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

	if foundTx != nil {
		if err = transaction.UpdateG(); err != nil {
			return transaction, err
		}
	} else {
		if err = transaction.InsertG(); err != nil {
			return nil, err
		}
	}

	return transaction, nil
}

func saveUpdateInputs(transaction *model.Transaction, jsonTx *lbrycrd.TxRawResult, txDbCrAddrMap *txDebitCredits) {
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
}

func saveUpdateOutputs(transaction *model.Transaction, jsonTx *lbrycrd.TxRawResult, txDbCrAddrMap *txDebitCredits) {
	vouts := jsonTx.Vout
	voutjobs := make(chan voutToProcess, len(vouts))
	errorchan := make(chan error, len(vouts))
	workers := util.Min(len(vouts), runtime.NumCPU())
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
}

func setSendReceive(transaction *model.Transaction, txDbCrAddrMap *txDebitCredits) {
	for addr, DC := range txDbCrAddrMap.addrDCMap {

		address := datastore.GetAddress(addr)

		txAddr := datastore.GetTxAddress(transaction.ID, address.ID)

		txAddr.CreditAmount = DC.Credits()
		txAddr.DebitAmount = DC.Debits()

		if err := datastore.PutTxAddress(txAddr); err != nil {
			panic(err) //Should never happen or something is wrong
		}
	}
}
