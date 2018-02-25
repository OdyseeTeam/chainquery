package daemon

import (
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/txscript"
	"github.com/lbryio/chainquery/app/lbrycrd"
	"github.com/lbryio/chainquery/app/model"
	"github.com/lbryio/errors.go"
	log "github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"golang.org/x/crypto/ripemd160"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var workers int = runtime.NumCPU() / 2 //Split cores between processors and lbycrd
var lastHeightProcess uint64 = 0       // Around 165,000 is when protobuf takes affect.
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
	lastBlock, _ := model.Blocks(boil.GetDB(), qm.OrderBy(model.BlockColumns.Height+" DESC"), qm.Limit(1)).One()
	if lastBlock != nil && lastBlock.Height > 100 {
		//lastHeightProcess = lastBlock.Height - 100 //Start 100 sooner just in case something happened.
	}
	go daemonIteration()
	log.Info("Daemon initialized and running")
	for {
		time.Sleep(1 * time.Second)
		if !running {
			go daemonIteration()
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
	next := lastHeightProcess + uint64(1)
	if *height >= next {
		go runBlockProcessing(&next)
	}
	if next%200 == 0 {
		log.Info("running iteration at block height ", next)
	}

	return nil
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
	foundBlock, _ := model.FindBlock(boil.GetDB(), jsonBlock.Hash)
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
	}
	Txs := jsonBlock.Tx
	for i := range Txs {
		jsonTx, err := lbrycrd.DefaultClient().GetRawTransactionResponse(Txs[i])
		err = processTx(jsonTx)
		if err != nil {
			log.Error(err)
		}
	}
	goToNextBlock(height)
}

func goToNextBlock(height *uint64) {
	lastHeightProcess = *height
	if lastHeightProcess+uint64(1) < blockHeight {
		daemonIteration()
	} else {
		running = false
	}
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
		}
	}
	vouts := jsonTx.Vout
	for i := range vouts {
		err := processVout(&vouts[i], &transaction.Hash)
		if err != nil {
			log.Error("Vout Error->", err, " - ", transaction.Hash)
		}
	}

	return err
}

func processVin(jsonVin *lbrycrd.Vin, txHash *string) error {
	vin := &model.Input{}
	//ID is txid + sequence
	inputid := *txHash + strconv.Itoa(int(jsonVin.Sequence))
	foundVin, err := model.FindInput(boil.GetDB(), inputid)
	if foundVin != nil {
		vin = foundVin
	}

	if jsonVin.Coinbase != "" {
		processCoinBaseVin(jsonVin)
	} else {
		vin.ID = inputid
		vin.TransactionID = *txHash
		vin.SequenceID = uint(jsonVin.Sequence)
		vin.PrevoutHash.String = jsonVin.Txid
		vin.PrevoutN.Uint = uint(jsonVin.Vout)
		vin.ScriptSigHex.String = jsonVin.ScriptSig.Hex
		vin.ScriptSigSSM.String = jsonVin.ScriptSig.Asm

	}
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

func processVout(jsonVout *lbrycrd.Vout, txHash *string) error {
	vout := &model.Output{}
	//ID is txid + sequence
	outputid := *txHash + strconv.Itoa(int(jsonVout.N))
	foundVout, err := model.FindOutput(boil.GetDB(), outputid)
	if foundVout != nil {
		vout = foundVout
	}

	vout.ID = outputid
	vout.TransactionID = *txHash
	vout.SequenceID = uint(jsonVout.N)
	vout.Value.String = strconv.Itoa(int(jsonVout.Value))
	vout.ScriptPubKeyAsm.String = jsonVout.ScriptPubKey.Asm
	vout.ScriptPubKeyHex.String = jsonVout.ScriptPubKey.Hex
	vout.Type.String = jsonVout.ScriptPubKey.Type
	scriptBytes, err := hex.DecodeString(vout.ScriptPubKeyHex.String)
	if err != nil {
		return err
	}
	isP2SH := txscript.IsPayToScriptHash(scriptBytes)
	isP2PK := vout.Type.String == "pubkey"
	isP2PKH := vout.Type.String == "pubkeyhash"
	isNonStandard := vout.Type.String == "nonstandard"
	if isP2SH {
		//log.Debug("Found pay to script hash outpoint")
	} else if isP2PK {
		//log.Debug("Found pay to pub key outpoint")
	} else if isP2PKH {
		//log.Debug("Found pay to pub key hash outpoint")
	} else if isNonStandard {
		err = processAsClaim(scriptBytes, *vout)
		if err != nil {
			return err
		}
	}
	return nil
}

func processCoinBaseVin(jsonVin *lbrycrd.Vin) {
	//log.Debug("Coinbase transaction")
}

func processAsClaim(script []byte, vout model.Output) error {
	if lbrycrd.IsClaimNameScript(script) {
		_, _, err := processClaimNameScript(&script, vout)
		if err != nil {
			return err
		}
		return nil
	} else if lbrycrd.IsClaimSupportScript(script) {
		_, _, err := processClaimSupportScript(&script)
		if err != nil {
			return err
		}
		return nil
	} else if lbrycrd.IsClaimUpdateScript(script) {
		_, _, err := processClaimUpdateScript(&script)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.Base("Not a claim -- " + hex.EncodeToString(script))
}

func processClaimNameScript(script *[]byte, vout model.Output) (name string, claimid string, err error) {
	name, value, _, err := lbrycrd.ParseClaimNameScript(*script)
	if err != nil {
		return name, claimid, err
	}
	claim, err := lbrycrd.DecodeClaimValue(name, value)
	if claim != nil {
		hasher := ripemd160.New()
		hasher.Write([]byte(vout.TransactionID + strconv.Itoa(int(vout.SequenceID))))
		hashBytes := hasher.Sum(nil)
		claimId := fmt.Sprintf("%x", hashBytes)
		log.Info("ClaimName ", name, " ClaimId ", claimId)
	}

	return name, claimid, err
}

func processClaimSupportScript(script *[]byte) (name string, claimid string, err error) {
	name, claimid, _, err = lbrycrd.ParseClaimSupportScript(*script)
	if err != nil {
		errors.Prefix("Claim support processing error: ", err)
	}
	log.Debug("ClaimSupport ", name, " ClaimId ", claimid)

	return name, claimid, err
}

func processClaimUpdateScript(script *[]byte) (name string, claimid string, err error) {
	return name, claimid, err
}
