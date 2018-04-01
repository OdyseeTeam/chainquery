package daemon

import (
	"encoding/json"
	"runtime"
	"strings"
	"time"

	p "github.com/lbryio/chainquery/daemon/processing"
	"github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/errors"

	log "github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

const (
	// Mode for processing speed
	BEASTMODE      = 0 // serialized until finished - new thread each daemon iteration.
	SLOWSTEADYMODE = 1 // 1 block per 100 ms
	DELAYMODE      = 2 // 1 block per delay
	DAEMONMODE     = 3 // 1 block per Daemon iteration
)

var workers int = runtime.NumCPU() / 2 //Split cores between processors and lbycrd
var lastHeightProcess uint64 = 0       // Around 165,000 is when protobuf takes affect.
var blockHeight uint64 = 0
var running bool = false

//Configuration
var ProcessingMode int            //Set in main on init
var ProcessingDelay time.Duration //Set by `applySettings`
var DaemonDelay time.Duration     //Set by `applySettings`
var iteration int64 = 0

func InitDaemon() {
	//testFunction()
	initBlockQueue()
	runDaemon()
}

func initBlockQueue() {

}

func testFunction(params ...interface{}) {

	names, err := lbrycrd.DefaultClient().GetClaimsInTrie()
	goodones := 0
	if err != nil {
		log.Error(err)
	} else {
		for i := range names {
			if goodones < 10 {
				name := names[i]
				for i := range name.Claims {
					claim := name.Claims[i]

					decodedValue := []byte(claim.Value)
					if err != nil {
						//log.Error(err)
						continue
					}
					decodedClaim, err := lbrycrd.DecodeClaimValue(name.Name, decodedValue)
					if err != nil {
						//log.Error(err)
						continue
					}
					println(name.Name, " - ", decodedClaim.GetStream().GetMetadata().GetTitle())
					jsonBytes, err := json.Marshal(*decodedClaim)
					if err != nil {
						//log.Error(err)
						continue
					}
					println(string(jsonBytes))
					goodones++
				}
			}
		}
	}
	//panic(errors.Base("only run test method"))
}

func ApplySettings(processingDelay time.Duration, daemonDelay time.Duration) {
	DaemonDelay = daemonDelay
	ProcessingDelay = processingDelay
	if ProcessingMode == BEASTMODE {
		ProcessingDelay = 0 * time.Millisecond
	} else if ProcessingMode == SLOWSTEADYMODE {
		ProcessingDelay = 100 * time.Millisecond
	} else if ProcessingMode == DELAYMODE {
		ProcessingDelay = processingDelay
	} else if ProcessingMode == DAEMONMODE {
		ProcessingDelay = daemonDelay //
	}
}

func runDaemon() func() {
	lastBlock, _ := model.Blocks(boil.GetDB(), qm.OrderBy(model.BlockColumns.Height+" DESC"), qm.Limit(1)).One()
	if lastBlock != nil && lastBlock.Height > 100 {
		lastHeightProcess = lastBlock.Height - 100 //Start 100 sooner just in case something happened.
	}
	go daemonIteration()
	log.Info("Daemon initialized and running")
	for {
		time.Sleep(DaemonDelay)
		if !running {
			log.Debug("Running daemon iteration ", iteration)
			go daemonIteration()
			iteration++
		}
	}
	return func() {}
}

func daemonIteration() error {

	height, err := lbrycrd.DefaultClient().GetBlockCount()
	if err != nil {
		return err
	}
	blockHeight = *height
	if lastHeightProcess == uint64(0) {
		runGenesisBlock()
	}
	next := lastHeightProcess + 1
	if blockHeight >= next {
		go runBlockProcessing(&next)
	}
	if next%50 == 0 {
		log.Info("running iteration at block height ", next, runtime.NumGoroutine(), " go routines")
	}

	return nil
}

func runGenesisBlock() {
	genesis := uint64(0)
	runBlockProcessing(&genesis)
}

func getBlockToProcess(height *uint64) (*lbrycrd.GetBlockResponse, error) {
	hash, err := lbrycrd.DefaultClient().GetBlockHash(*height)
	if err != nil {
		return nil, errors.Prefix("GetBlockHash Error("+string(*height)+"): ", err)
	}
	jsonBlock, err := lbrycrd.DefaultClient().GetBlock(*hash)
	if err != nil {
		return nil, errors.Prefix("GetBlock Error("+*hash+"): ", err)
	}
	return jsonBlock, nil
}

func runBlockProcessing(height *uint64) {
	//defer util.TimeTrack(time.Now(), "runBlockProcessing")
	running = true
	jsonBlock, err := getBlockToProcess(height)
	if err != nil {
		log.Error("Get Block Error: ", err)
		goToNextBlock(height)
		return
	}
	block := &model.Block{}
	foundBlock, _ := model.BlocksG(qm.Where(model.BlockColumns.Hash+"=?", jsonBlock.Hash)).One()
	if foundBlock != nil {
		block = foundBlock
	}
	block.Height = uint64(*height)
	block.Confirmations = uint(jsonBlock.Confirmations)
	block.Hash = jsonBlock.Hash
	block.BlockTime = uint64(jsonBlock.Time)
	block.Bits = jsonBlock.Bits
	block.BlockSize = uint64(jsonBlock.Size)
	block.Chainwork = jsonBlock.ChainWork
	block.Difficulty = jsonBlock.Difficulty
	block.MerkleRoot = jsonBlock.MerkleRoot
	block.NameClaimRoot = jsonBlock.NameClaimRoot
	block.NextBlockHash.String = jsonBlock.NextHash
	block.PreviousBlockHash.String = jsonBlock.PreviousHash
	block.TransactionHashes.String = strings.Join(jsonBlock.Tx, ",")
	block.Version = uint64(jsonBlock.Version)
	block.VersionHex = jsonBlock.VersionHex
	if foundBlock != nil {
		err = block.Update(boil.GetDB())
	} else {
		err = block.Insert(boil.GetDB())
	}
	if err != nil {
		log.Error(err)
	}
	Txs := jsonBlock.Tx
	for i := range Txs {
		jsonTx, err := lbrycrd.DefaultClient().GetRawTransactionResponse(Txs[i])
		err = processTx(jsonTx, block.BlockTime)
		if err != nil {
			log.Error(err)
		}
	}
	goToNextBlock(height)
}

func goToNextBlock(height *uint64) {
	lastHeightProcess = *height
	workToDo := lastHeightProcess+uint64(1) < blockHeight && lastHeightProcess != 0
	if workToDo {
		time.Sleep(ProcessingDelay)
		go daemonIteration()
	} else {
		running = false
	}
}

func processTx(jsonTx *lbrycrd.TxRawResult, blockTime uint64) error {
	//defer util.TimeTrack(time.Now(), "processTx "+jsonTx.Txid+" -- ")
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

	_, err = p.CreateUpdateAddresses(jsonTx.Vout, blockTime)
	if err != nil {
		err := errors.Prefix("Address Creation Error: ", err)
		return err
	}

	txDbCrAddrMap := p.NewTxDebitCredits()

	if foundTx != nil {
		transaction.Update(boil.GetDB())
	} else {
		err = transaction.Insert(boil.GetDB())
	}
	if err != nil {
		return err
	}
	vins := jsonTx.Vin
	for i := range vins {
		err = p.ProcessVin(&vins[i], *transaction, txDbCrAddrMap)
		if err != nil {
			log.Error("Vin Error->", err)
			panic(err)
		}
	}
	vouts := jsonTx.Vout
	for i := range vouts {
		err := p.ProcessVout(&vouts[i], *transaction, txDbCrAddrMap)
		if err != nil {
			log.Error("Vout Error->", err, " - ", transaction.Hash)
			panic(err)
		}
	}
	for addr, DC := range txDbCrAddrMap.AddrDCMap {

		address := datastore.GetAddress(addr)

		txAddr := datastore.GetTxAddress(transaction.ID, address.ID)

		txAddr.CreditAmount = DC.Credits()
		txAddr.DebitAmount = DC.Credits()

		datastore.PutTxAddress(txAddr)

	}

	return err
}
