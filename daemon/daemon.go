package daemon

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/errors.go"

	log "github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"golang.org/x/crypto/ripemd160"
)

var blockQueue = make(chan int)
var queuedHeightMutex sync.Mutex
var lastQueuedHeight int

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

func runDaemon() {
	lastBlock, _ := model.Blocks(boil.GetDB(), qm.OrderBy(model.BlockColumns.Height+" DESC"), qm.Limit(1)).One()
	if lastBlock != nil && lastBlock.Height > 100 {
		queuedHeightMutex.Lock()
		lastQueuedHeight = int(lastBlock.Height) - 100 // Start 100 sooner just in case something happened.
		queuedHeightMutex.Unlock()
	}

	log.Info("Daemon initialized and running")

	// create worker
	go func() {
		for {
			height := <-blockQueue
			runBlockProcessing(height)
		}
	}()

	// queue blocks
	queueBlocks()
}

func queueBlocks() {
	for {
		height, err := lbrycrd.DefaultClient().GetBlockCount()
		if err != nil {
			log.Errorln(err)
		}
		blockHeight := int(*height)

		queuedHeightMutex.Lock()
		if blockHeight > lastQueuedHeight {
			if blockHeight%200 == 0 {
				log.Info("queued block ", blockHeight)
			}
			lastQueuedHeight++
			queuedHeightMutex.Unlock()
			blockQueue <- blockHeight
			continue
		}

		time.Sleep(10 * time.Second)
	}
}

func getBlockToProcess(height int) (*lbrycrd.GetBlockResponse, error) {
	hash, err := lbrycrd.DefaultClient().GetBlockHash(uint64(height))
	if err != nil {
		return nil, errors.Prefix("GetBlockHash Error("+strconv.Itoa(height)+"): ", err)
	}
	jsonBlock, err := lbrycrd.DefaultClient().GetBlock(*hash)
	if err != nil {
		return nil, errors.Prefix("GetBlock Error("+*hash+"): ", err)
	}
	return jsonBlock, nil
}

func runBlockProcessing(height int) {
	jsonBlock, err := getBlockToProcess(height)
	if err != nil {
		log.Error("Get Block Error: ", err)
		return
	}
	block := &model.Block{}
	foundBlock, _ := model.BlocksG(qm.Where(model.BlockColumns.Hash+"=?", jsonBlock.Hash)).One()
	if foundBlock != nil {
		block = foundBlock
	}
	block.Height = uint64(height)
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
		err = processTx(jsonTx)
		if err != nil {
			log.Error(err)
		}
	}
}

func processTx(jsonTx *lbrycrd.TxRawResult) error {
	transaction := &model.Transaction{}
	foundTx, err := model.TransactionsG(qm.Where(model.TransactionColumns.Hash+"=?", jsonTx.Txid)).One()
	if foundTx != nil {
		transaction = foundTx
	}
	transaction.Hash = jsonTx.Txid
	transaction.Version = int(jsonTx.Version)
	transaction.BlockByHashID.String = jsonTx.BlockHash
	transaction.CreatedTime = time.Unix(0, jsonTx.Blocktime)
	transaction.TransactionTime.Uint64 = uint64(jsonTx.Blocktime)
	transaction.LockTime = uint(jsonTx.LockTime)
	transaction.InputCount = uint(len(jsonTx.Vin))
	transaction.OutputCount = uint(len(jsonTx.Vout))
	transaction.Raw.String = jsonTx.Hex
	transaction.TransactionSize = uint64(jsonTx.Size)
	totalValue := float64(0)
	for i := range jsonTx.Vout {
		totalValue = totalValue + jsonTx.Vout[i].Value
	}
	transaction.Value = strconv.FormatFloat(totalValue, 'f', -1, 64)
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
		err = processVin(&vins[i], &transaction.ID)
		if err != nil {
			log.Error("Vin Error->", err)
		}
	}
	vouts := jsonTx.Vout
	for i := range vouts {
		err := processVout(&vouts[i], &transaction.ID)
		if err != nil {
			log.Error("Vout Error->", err, " - ", transaction.Hash)
		}
	}

	return err
}

func processVin(jsonVin *lbrycrd.Vin, txId *uint64) error {
	vin := &model.Input{}
	foundVin, err := model.InputsG(
		qm.Where(model.InputColumns.TransactionID+"=?", txId),
		qm.And(model.InputColumns.Sequence+"=?", jsonVin.Sequence)).One() //boil.GetDB(), inputid)
	if foundVin != nil {
		vin = foundVin
	}

	if jsonVin.Coinbase != "" {
		processCoinBaseVin(jsonVin)
	} else {
		vin.TransactionID = *txId
		vin.Sequence.Uint = uint(jsonVin.Sequence)
		vin.PrevoutHash.String = jsonVin.Txid
		vin.PrevoutN.Uint = uint(jsonVin.Vout)
		vin.ScriptSigHex.String = jsonVin.ScriptSig.Hex
		vin.ScriptSigAsm.String = jsonVin.ScriptSig.Asm

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

func processVout(jsonVout *lbrycrd.Vout, txId *uint64) error {
	vout := &model.Output{}
	foundVout, err := model.OutputsG(
		qm.Where(model.OutputColumns.TransactionID+"=?", txId),
		qm.And(model.OutputColumns.Vout+"=?", jsonVout.N)).One() //boil.GetDB(), outputid)
	if foundVout != nil {
		vout = foundVout
	}

	vout.TransactionID = *txId
	vout.Vout.Uint = uint(jsonVout.N)
	vout.Value.String = strconv.Itoa(int(jsonVout.Value))
	vout.ScriptPubKeyAsm.String = jsonVout.ScriptPubKey.Asm
	vout.ScriptPubKeyHex.String = jsonVout.ScriptPubKey.Hex
	vout.Type.String = jsonVout.ScriptPubKey.Type
	jsonAddresses, err := json.Marshal(jsonVout.ScriptPubKey.Addresses)
	if err != nil {
		log.Error("Could not marshall address list of Vout")
		err = nil //reset error
	} else {
		vout.AddressList.String = string(jsonAddresses)
	}
	scriptBytes, err := hex.DecodeString(vout.ScriptPubKeyHex.String)
	if err != nil {
		return err
	}
	isP2SH := vout.Type.String == "scripthash"
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
		_, _, err := processClaimSupportScript(&script, vout)
		if err != nil {
			return err
		}
		return nil
	} else if lbrycrd.IsClaimUpdateScript(script) {
		_, _, err := processClaimUpdateScript(&script, vout)
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
		errors.Prefix("Claim name processing error: ", err)
		return name, claimid, err
	}
	_, err = lbrycrd.DecodeClaimValue(name, value)
	if false { //claim != nil {
		hasher := ripemd160.New()
		value := strconv.Itoa(int(vout.TransactionID)) + strconv.Itoa(int(vout.Vout.Uint))
		hasher.Write([]byte(value))
		hashBytes := hasher.Sum(nil)
		claimId := fmt.Sprintf("%x", hashBytes)
		if claimId != "" {
			//log.Debug("ClaimName ", name, " ClaimId ", claimId)
		}
	}

	return name, claimid, err
}

func processClaimSupportScript(script *[]byte, vout model.Output) (name string, claimid string, err error) {
	name, claimid, _, err = lbrycrd.ParseClaimSupportScript(*script)
	if err != nil {
		errors.Prefix("Claim support processing error: ", err)
		return name, claimid, err
	}
	//log.Debug("ClaimSupport ", name, " ClaimId ", claimid)

	return name, claimid, err
}

func processClaimUpdateScript(script *[]byte, vout model.Output) (name string, claimId string, err error) {
	name, claimId, value, _, err := lbrycrd.ParseClaimUpdateScript(*script)
	if err != nil {
		errors.Prefix("Claim update processing error: ", err)
		return name, claimId, err
	}
	claim, err := lbrycrd.DecodeClaimValue(name, value)
	if claim != nil {
		//log.Debug("ClaimUpdate ", name, " ClaimId ", claimId)
	}
	return name, claimId, err
}
