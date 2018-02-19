package daemon

import (
	"github.com/lbryio/chainquery/app/lbrycrd"
	"github.com/lbryio/chainquery/app/model"
	log "github.com/sirupsen/logrus"
	"runtime"
	"time"
)

var workers int = runtime.NumCPU() / 2 //Split cores between processors and lbycrd
var lastHeightProcess = 0

func InitDaemon() {
	initBlockQueue()
	runDaemon()
}

func initBlockQueue() {
	//
}

func runDaemon() func() {
	log.Info("Daemon initialized and running")
	for {
		time.Sleep(10 * time.Second)
		go daemonIteration()
	}
	return func() {}
}

func daemonIteration() error {
	blockHeight, err := lbrycrd.DefaultClient().GetBlockCount()
	if err != nil {
		log.Error("Iteration Error:", err)
		return err
	}
	log.Info("running iteration at block height ", blockHeight)
	go runBlockProcessing(blockHeight)
	return nil
}

func runBlockProcessing(height int64) {
	jsonBlock, err := getBlockToProcess(height)
	if err != nil {
		log.Error("Block Processing Error: ", err)
		return
	}
	block := &model.Block{}
	block.Height = uint64(height)
	Txs := jsonBlock.Tx
	for i := range Txs {
		jsonTx, err := lbrycrd.DefaultClient().GetRawTransactionResponse(Txs[i])
		tx, err := processTx(jsonTx)
		if err != nil {
			return
		}
		println("Transaction", Txs[i])
		block.AddTransactionsG(false, tx)
	}
	block.InsertG()
}

func processTx(jsonTx *lbrycrd.TxRawResult) (*model.Transaction, error) {
	transaction := &model.Transaction{}
	transaction.Hash = jsonTx.Hash
	return transaction, nil
}

func getBlockToProcess(height int64) (*lbrycrd.GetBlockResponse, error) {
	hash, err := lbrycrd.DefaultClient().GetBlockHash(height)
	if err != nil {
		log.Error("GetBlockHash Error: ", err)
		return nil, err
	}
	jsonBlock, err := lbrycrd.DefaultClient().GetBlock(hash)
	if err != nil {
		log.Error("GetBlock Error: ", hash, err)
		return nil, err
	}
	return jsonBlock, nil
}
