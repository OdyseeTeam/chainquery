package daemon

import (
	"github.com/lbryio/chainquery/app/lbrycrd"
	"github.com/lbryio/chainquery/app/model"
	"github.com/lbryio/lbryschema.go/claim"
	"github.com/lbryio/lbryschema.go/pb"
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
		//daemonIteration()
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
	if err != nil {
		return err
	}
	vins := jsonTx.Vin
	for i := range vins {
		err = processVin(&vins[i], &transaction.Hash)
		if err != nil {
			log.Error("Vin Error->", err)
			err = nil
		}
	}
	vouts := jsonTx.Vout
	for i := range vouts {
		err := processVout(&vouts[i])
		if err != nil {
			log.Error("Vout Error->", err)
			err = nil
		}
	}

	return err
}

func processVin(jsonVin *lbrycrd.Vin, txHash *string) error {
	vin := &model.Input{}
	inputid := *txHash + strconv.Itoa(int(jsonVin.Sequence))
	foundVin, err := model.FindInput(boil.GetDB(), inputid)
	if foundVin != nil {
		vin = foundVin
	}
	vin.ID = inputid
	vin.TransactionID = *txHash
	vin.SequenceID = uint(jsonVin.Sequence)
	vin.Coinbase.String = jsonVin.Coinbase
	vin.PrevoutHash.String = jsonVin.Txid
	vin.PrevoutN.Uint = uint(jsonVin.Vout)
	println("Nil ScriptSiq", jsonVin.ScriptSig == nil)
	processScript(&jsonVin.ScriptSig.Hex)
	//vin.ScriptSigHex.String = jsonVin.ScriptSig.Hex
	//vin.ScriptSigSSM.String = jsonVin.ScriptSig.Asm
	//ForeignKey
	err = nil //reset to catch error for update/insert
	if foundVin != nil {
		//err = vin.Update(boil.GetDB())
	} else {
		//err = vin.Insert(boil.GetDB())
	}
	if err != nil {
		return err
	}
	return nil
}

func processVout(jsonVout *lbrycrd.Vout) error {
	return nil
}

func processScript(hex *string) {

	c := new(claim.ClaimHelper)
	c.Claim = new(pb.Claim)
	log.Debug(c.String())
	err := c.LoadFromHexString(*hex, "lbrycrd_main")
	if err != nil {
		log.Error(err)
	} else {
		log.Info("Sucess")
	}
}
