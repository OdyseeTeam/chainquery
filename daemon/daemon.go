package daemon

import (
	"runtime"
	"strings"
	"time"

	"github.com/lbryio/chainquery/daemon/processing"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/errors"

	"github.com/lbryio/chainquery/daemon/upgrademanager"
	"github.com/lbryio/chainquery/util"
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
var Reindex bool = false
var BlockConfirmationBuffer int64 = 6 //Block is accepted at 6 confirmations

//Configuration
var ProcessingMode int            //Set in main on init
var ProcessingDelay time.Duration //Set by `applySettings`
var DaemonDelay time.Duration     //Set by `applySettings`
var iteration int64 = 0

func InitDaemon() {
	//testFunction()
	upgrademanager.RunUpgradesForVersion()
	initBlockQueue()
	runDaemon()
}

func initBlockQueue() {

}

func testFunction(params ...interface{}) {

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
	if lastBlock != nil && lastBlock.Height > 100 && !Reindex {
		lastHeightProcess = lastBlock.Height - 100 //Start 100 sooner just in case something happened.
	}
	log.Info("Daemon initialized and running")
	for {
		time.Sleep(DaemonDelay)
		if !running {
			running = true
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
	defer util.TimeTrack(time.Now(), "runBlockProcessing", "daemonprofile")
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
		err = processing.ProcessTx(jsonTx, block.BlockTime)
		if err != nil {
			log.Error(err)
		}
	}
	goToNextBlock(height)
}

func goToNextBlock(height *uint64) {
	defer util.TimeTrack(time.Now(), "gotonextblock", "daemonprofile")
	lastHeightProcess = *height
	workToDo := lastHeightProcess+uint64(1) < blockHeight && lastHeightProcess != 0
	if workToDo {
		time.Sleep(ProcessingDelay)
		go daemonIteration()
	} else {
		running = false
	}
}
