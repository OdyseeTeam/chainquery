package daemon

import (
	"github.com/lbryio/chainquery/app/lbrycrd"
	"github.com/lbryio/chainquery/app/model"
	"github.com/lbryio/sqlboiler/boil"
	log "github.com/sirupsen/logrus"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var workers int = runtime.NumCPU() / 2 //Split cores between processors and lbycrd
var lastHeightProcess uint64 = 0
var blockHeight uint64 = 0
var running bool = false

func InitDaemon() {
	initBlockQueue()
	runDaemon()
}

func initBlockQueue() {
	//
}

func runDaemon() func() {
	go daemonIteration()
	log.Info("Daemon initialized and running")
	for {
		time.Sleep(10 * time.Second)
		go daemonIteration()
	}
	return func() {}
}

func daemonIteration() error {

	height, err := lbrycrd.DefaultClient().GetBlockCount()
	if err != nil {
		log.Error("Iteration Error:", err)
		return err
	}
	blockHeight = *height
	next := lastHeightProcess + uint64(1)
	if *height >= next {
		go runBlockProcessing(&next)
	}
	log.Info("running iteration at block height ", *height)

	return nil
}

func runBlockProcessing(height *uint64) {
	running = true
	jsonBlock, err := getBlockToProcess(height)
	if err != nil {
		log.Error("Block Processing Error: ", err)
		return
	}
	block := &model.Block{}
	foundBlock, err := model.FindBlock(boil.GetDB(), jsonBlock.Hash)
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
	block.NextBlockID.String = jsonBlock.NextHash
	block.PreviousBlockID.String = jsonBlock.PreviousHash
	block.TransactionHashes = strings.Join(jsonBlock.Tx, ",")
	block.Version = uint64(jsonBlock.Version)
	block.VersionHex = jsonBlock.VersionHex
	if foundBlock != nil {
		err = block.Update(boil.GetDB())
	} else {
		err = block.Insert(boil.GetDB())
	}
	if err != nil {
		log.Error(err)
		return
	}
	Txs := jsonBlock.Tx
	for i := range Txs {
		jsonTx, err := lbrycrd.DefaultClient().GetRawTransactionResponse(Txs[i])
		err = processTx(jsonTx)
		if err != nil {
			log.Error(err)
		}
	}

	lastHeightProcess = block.Height
	if lastHeightProcess+uint64(1) < blockHeight {
		daemonIteration()
	}
	running = false
}

func processTx(jsonTx *lbrycrd.TxRawResult) error {
	transaction := &model.Transaction{}
	foundTx, err := model.FindTransaction(boil.GetDB(), jsonTx.Txid)
	if foundTx != nil {
		transaction = foundTx
	}
	transaction.Hash = jsonTx.Txid
	transaction.Version = int(jsonTx.Version)
	transaction.BlockID = jsonTx.BlockHash
	transaction.CreatedTime = uint(jsonTx.Blocktime)
	transaction.TransactionTime.Uint64 = uint64(jsonTx.Blocktime)
	transaction.LockTime = uint(jsonTx.LockTime)
	transaction.InputCount = uint(len(jsonTx.Vin))
	transaction.OutputCount = uint(len(jsonTx.Vout))
	transaction.Raw.String = jsonTx.Hex
	transaction.TransactionSize = uint64(jsonTx.Size)
	if foundTx != nil {
		transaction.Update(boil.GetDB())
	} else {
		err = transaction.Insert(boil.GetDB())
	}

	return err
}

func getBlockToProcess(height *uint64) (*lbrycrd.GetBlockResponse, error) {
	hash, err := lbrycrd.DefaultClient().GetBlockHash(*height)
	if err != nil {
		log.Error("GetBlockHash Error: ", err)
		return nil, err
	}
	jsonBlock, err := lbrycrd.DefaultClient().GetBlock(*hash)
	if err != nil {
		log.Error("GetBlock Error: ", *hash, err)
		return nil, err
	}
	return jsonBlock, nil
}
