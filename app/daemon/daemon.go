package daemon

import (
	"encoding/json"
	p "github.com/lbryio/chainquery/app/daemon/processing"
	"github.com/lbryio/chainquery/app/lbrycrd"
	"github.com/lbryio/chainquery/app/model"
	"github.com/lbryio/errors.go"
	log "github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	// Mode for processing speed
	BEASTMODE      = 0 // serialized until finished - new thread each daemon iteration.
	SLOWSTEADYMODE = 1 // 1 block per 100 ms
	DELAYMODE      = 2 // 1 block per delay
	DAEMONMODE     = 3 // 1 block per Daemon iteration
	// Default Delays
	PROCDELAY   = 50 * time.Millisecond
	DAEMONDELAY = 1000 * time.Millisecond
)

var workers int = runtime.NumCPU() / 2 //Split cores between processors and lbycrd
var lastHeightProcess uint64 = 3200    // Around 165,000 is when protobuf takes affect.
var blockHeight uint64 = 0
var running bool = false
var ProcessingMode int            //Set in main on init
var processingDelay time.Duration //Set by `applySettings`
var iteration int64 = 0

func InitDaemon() {
	applySettings()
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

func applySettings() {
	if ProcessingMode == BEASTMODE {
		processingDelay = 0 * time.Millisecond
	} else if ProcessingMode == SLOWSTEADYMODE {
		processingDelay = 100 * time.Millisecond
	} else if ProcessingMode == DELAYMODE {
		processingDelay = PROCDELAY
	} else if ProcessingMode == DAEMONMODE {
		processingDelay = DAEMONDELAY
	}
}

func runDaemon() func() {
	lastBlock, _ := model.Blocks(boil.GetDB(), qm.OrderBy(model.BlockColumns.Height+" DESC"), qm.Limit(1)).One()
	if lastBlock != nil && lastBlock.Height > 100 {
		//lastHeightProcess = lastBlock.Height - 100 //Start 100 sooner just in case something happened.
	}
	go daemonIteration()
	log.Info("Daemon initialized and running")
	for {
		time.Sleep(DAEMONDELAY)
		if !running {
			log.Debug("Running iteration ", iteration)
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
	if next%200 == 0 {
		log.Info("running iteration at block height ", next)
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
	block.Difficulty = strconv.FormatFloat(jsonBlock.Difficulty, 'f', -1, 64)
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
	workToDo := lastHeightProcess+uint64(1) < blockHeight &&
		lastHeightProcess != 0
	if workToDo {
		time.Sleep(processingDelay)
		daemonIteration()
	} else {
		running = false
	}
}

func processTx(jsonTx *lbrycrd.TxRawResult, blockTime uint64) error {
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
	transaction.Value = strconv.FormatFloat(0.0, 'f', -1, 64) //strconv.FormatFloat(p.GetTotalValue(jsonTx.Vout), 'f', -1, 64)

	_, err = p.CreateUpdateAddresses(jsonTx.Vout, blockTime)
	if err != nil {
		err := errors.Prefix("Address Creation Error: ", err)
		return err
	}

	txDbCrAddrMap := map[string]p.AddrDebitCredits{}

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
		err = p.ProcessVin(&vins[i], &transaction.ID, jsonTx.Txid, txDbCrAddrMap)
		if err != nil {
			log.Error("Vin Error->", err)
			panic(err)
		}
	}
	vouts := jsonTx.Vout
	for i := range vouts {
		err := p.ProcessVout(&vouts[i], &transaction.ID, transaction.Hash, txDbCrAddrMap)
		if err != nil {
			log.Error("Vout Error->", err, " - ", transaction.Hash)
			panic(err)
		}
	}

	return err
}
