package processing

import (
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/chainquery/util"
	"github.com/lbryio/lbry.go/errors"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"strings"
	"time"
)

func RunBlockProcessing(height *uint64) {
	defer util.TimeTrack(time.Now(), "runBlockProcessing", "daemonprofile")
	jsonBlock, err := getBlockToProcess(height)
	if err != nil {
		logrus.Error("Get Block Error: ", err)
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
		err = block.UpdateG()
	} else {
		err = block.InsertG()
	}
	if err != nil {
		logrus.Error(err)
	}
	Txs := jsonBlock.Tx
	for i := range Txs {
		jsonTx, err := lbrycrd.GetRawTransactionResponse(Txs[i])
		err = ProcessTx(jsonTx, block.BlockTime)
		if err != nil {
			logrus.Error(err)
		}
	}
}

func getBlockToProcess(height *uint64) (*lbrycrd.GetBlockResponse, error) {
	hash, err := lbrycrd.GetBlockHash(*height)
	if err != nil {
		return nil, errors.Prefix("GetBlockHash Error("+string(*height)+"): ", err)
	}
	jsonBlock, err := lbrycrd.GetBlock(*hash)
	if err != nil {
		return nil, errors.Prefix("GetBlock Error("+*hash+"): ", err)
	}
	return jsonBlock, nil
}
