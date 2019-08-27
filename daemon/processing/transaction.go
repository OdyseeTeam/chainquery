package processing

import (
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/chainquery/util"
	"github.com/lbryio/lbry.go/extras/errors"

	"github.com/lbryio/lbry.go/extras/stop"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

type txToProcess struct {
	tx          *lbrycrd.TxRawResult
	blockTime   uint64
	blockHeight uint64
	failcount   int
}

type txProcessResult struct {
	tx          *lbrycrd.TxRawResult
	blockTime   uint64
	blockHeight uint64
	err         error
	failcount   int
}

func initTxWorkers(s *stop.Group, nrWorkers int, jobs <-chan txToProcess, results chan<- txProcessResult) {
	for i := 0; i < nrWorkers; i++ {
		s.Add(1)
		go func(worker int) {
			defer s.Done()
			txProcessor(s, jobs, results, worker)
			q(strconv.Itoa(worker) + " - WORKER TX - Finished all jobs")
		}(i)
	}
}

func txProcessor(s *stop.Group, jobs <-chan txToProcess, results chan<- txProcessResult, worker int) {
	for {
		select {
		case <-s.Ch():
			return
		case job := <-jobs:
			q(strconv.Itoa(worker) + " - WORKER TX - Start new job " + job.tx.Txid)
			err := ProcessTx(job.tx, job.blockTime, job.blockHeight)
			result := txProcessResult{
				tx:          job.tx,
				blockTime:   job.blockTime,
				blockHeight: job.blockHeight,
				err:         err,
				failcount:   job.failcount + 1}
			q(strconv.Itoa(worker) + " - WORKER TX - Finished new job " + job.tx.Txid)
			select {
			case <-s.Ch():
				q(strconv.Itoa(worker) + " - WORKER TX - discard finished job and stop " + job.tx.Txid)
				return
			default:
				q(strconv.Itoa(worker) + " - WORKER TX - Start sending result of job " + job.tx.Txid)
				results <- result
				q(strconv.Itoa(worker) + " - WORKER TX - End sending result of job " + job.tx.Txid)
			}
		}
	}
}

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
func ProcessTx(jsonTx *lbrycrd.TxRawResult, blockTime uint64, blockHeight uint64) error {
	defer util.TimeTrack(time.Now(), "processTx "+jsonTx.Txid+" -- ", "daemonprofile")

	//Save transaction before the id is used any where else otherwise it will be 0
	transaction, err := saveUpdateTransaction(jsonTx)
	if err != nil {
		return err
	}

	txDbCrAddrMap := newTxDebitCredits()

	_, err = createUpdateVoutAddresses(transaction, &jsonTx.Vout, blockTime)
	if err != nil {
		return errors.Prefix("Vout Address Creation Error: ", err)
	}
	_, err = createUpdateVinAddresses(transaction, &jsonTx.Vin, blockTime)
	if err != nil {
		return errors.Prefix("Vin Address Creation Error: ", err)
	}

	// Process the inputs of the tranasction
	err = saveUpdateInputs(transaction, jsonTx, txDbCrAddrMap)
	if err != nil {
		return err
	}

	// Process the outputs of the transaction
	err = saveUpdateOutputs(transaction, jsonTx, txDbCrAddrMap, blockHeight)
	if err != nil {
		return err
	}
	//Set the send and receive values for the transaction
	err = setSendReceive(transaction, txDbCrAddrMap)
	if err != nil {
		return err
	}

	return nil
}

func saveUpdateTransaction(jsonTx *lbrycrd.TxRawResult) (*model.Transaction, error) {
	transaction := &model.Transaction{}
	// Error is not helpful. It returns an error if there is nothing in the database.
	foundTx, _ := model.Transactions(qm.Where(model.TransactionColumns.Hash+"=?", jsonTx.Txid)).OneG()
	if foundTx != nil {
		transaction = foundTx
	}
	transaction.Hash = jsonTx.Txid
	transaction.Version = int(jsonTx.Version)
	transaction.BlockHashID.SetValid(jsonTx.BlockHash)
	transaction.CreatedTime = time.Unix(jsonTx.Blocktime, 0)
	transaction.TransactionTime.SetValid(uint64(jsonTx.Time))
	transaction.LockTime = uint(jsonTx.LockTime)
	transaction.InputCount = uint(len(jsonTx.Vin))
	transaction.OutputCount = uint(len(jsonTx.Vout))
	transaction.Raw.String = jsonTx.Hex
	transaction.TransactionSize = uint64(jsonTx.Size)

	if foundTx != nil {
		if err := transaction.UpdateG(boil.Infer()); err != nil {
			return transaction, err
		}
	} else {
		if err := transaction.InsertG(boil.Infer()); err != nil {
			return nil, err
		}
	}

	return transaction, nil
}

func saveUpdateInputs(transaction *model.Transaction, jsonTx *lbrycrd.TxRawResult, txDbCrAddrMap *txDebitCredits) error {
	vins := jsonTx.Vin
	vinjobs := make(chan vinToProcess)
	errorchan := make(chan error)
	workers := util.Min(len(vins), runtime.NumCPU())
	sQ := stop.New(nil)
	initVinWorkers(sQ, workers, vinjobs, errorchan)
	// Queue
	q("VIN SYNC started")
	sQ.Add(1)
	go func() {
		defer sQ.Done()
		//q("VIN start queueing")
		for i := range vins {
			select {
			case <-sQ.Ch():
				return
			default:
				//q("VIN start passing new job")
				vinjobs <- vinToProcess{jsonVin: &vins[i], tx: transaction, txDC: txDbCrAddrMap, vin: uint64(i)}
				//q("VIN end pass new job")
			}
		}
		//q("VIN end queueing")
		close(vinjobs)
	}()

	//Error check
	leftToProcess := len(vins)
	for err := range errorchan {
		leftToProcess--
		if err != nil {
			q("VIN error..stopping")
			sQ.StopAndWait()
			q("VIN error..stopped")
			return errors.Prefix("Vin Error->", err)
		}
		q("VIN processing..." + strconv.Itoa(leftToProcess))
		if leftToProcess == 0 {
			q("VIN stopping...")
			sQ.StopAndWait()
			q("VIN stopped...")
			q("VIN returning")
			return nil
		}
		continue
	}
	q("VIN SYNC ended")
	return nil
}

func saveUpdateOutputs(transaction *model.Transaction, jsonTx *lbrycrd.TxRawResult, txDbCrAddrMap *txDebitCredits, blockHeight uint64) error {
	vouts := jsonTx.Vout
	workers := util.Min(len(vouts), runtime.NumCPU())
	voutjobs := make(chan voutToProcess)
	errorchan := make(chan error)
	sQ := stop.New(nil)
	initVoutWorkers(sQ, workers, voutjobs, errorchan)
	// Queue
	q("VOUT SYNC started")
	sQ.Add(1)
	go func() {
		defer sQ.Done()
		q("VOUT start queueing")
		for i := range vouts {
			select {
			case <-sQ.Ch():
				return
			default:
				q("VOUT start passing new job")
				voutjobs <- voutToProcess{jsonVout: &vouts[i], tx: transaction, txDC: txDbCrAddrMap, blockHeight: blockHeight}
				q("VOUT end pass new job")
			}
		}
		q("VOUT SYNC finished")
		close(voutjobs)
	}()

	//Error check
	leftToProcess := len(vouts)
	for err := range errorchan {
		leftToProcess--
		if err != nil {
			q("VOUT error..stopping")
			sQ.StopAndWait()
			q("VOUT error..stopped")
			return errors.Prefix("Vout Error->", err)
		}
		if leftToProcess == 0 {
			q("VOUT stopping...")
			sQ.StopAndWait()
			q("VOUT stopped")
			q("VOUT returning")
			return nil
		}
		continue
	}
	q("VOUT SYNC ended")
	return nil
}

func setSendReceive(transaction *model.Transaction, txDbCrAddrMap *txDebitCredits) error {
	for addr, DC := range txDbCrAddrMap.addrDCMap {

		address := datastore.GetAddress(addr)

		txAddr := datastore.GetTxAddress(transaction.ID, address.ID)

		txAddr.CreditAmount = DC.Credits()
		txAddr.DebitAmount = DC.Debits()

		if err := datastore.PutTxAddress(txAddr); err != nil {
			return err //Should never happen or something is wrong
		}
	}
	return nil
}
