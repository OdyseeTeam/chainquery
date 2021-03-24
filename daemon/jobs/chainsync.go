package jobs

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lbryio/chainquery/daemon/processing"
	"github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/global"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/metrics"
	"github.com/lbryio/chainquery/model"

	"github.com/lbryio/lbry.go/extras/errors"
	"github.com/lbryio/lbryschema.go/claim"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

var chainSyncRunning = false
var chainSync *chainSyncStatus

// ChainSyncRunDuration specifies the duration, in seconds, the chain sync job will run at a time before stopping and
// storing state. It will get triggered periodically.
var ChainSyncRunDuration int

// ChainSyncDelay Specifies the duration, in milliseconds, between each block it synchronizes. Depending on the usage of
//the database you will want to add some delay between blocks so it does not overload the db server.
var ChainSyncDelay int

const chainSyncJob = "chainsync"

//ChainSyncAsync triggers the chain sync job in the background and returns
func ChainSyncAsync() {
	if !chainSyncRunning {
		chainSyncRunning = true
		go ChainSync()
	}
}

func endChainSync() {
	chainSyncRunning = false
	if r := recover(); r != nil {
		logrus.Error("Recovered From: ", r)
	}
}

// ChainSync synchronizes the chain data when it does not match lbrycrd. It runs for x duration before it stores state.
func ChainSync() {
	metrics.JobLoad.WithLabelValues("chain_sync").Inc()
	defer metrics.JobLoad.WithLabelValues("chain_sync").Dec()
	defer metrics.Job(time.Now(), "chain_sync")
	defer endChainSync()
	if chainSync == nil {
		chainSync = &chainSyncStatus{}
	}

	job, err := getChainSyncJobStatus()
	if err != nil {
		logrus.Error(err)
		saveJobError(job, err)
		return
	}

	if chainSync.LastHeight >= chainSync.MaxHeightStored {
		err := chainSync.updateMaxHeightStored()
		if err != nil {
			saveJobError(job, err)
			logrus.Error(err)
			return
		}
	}

	timeLimit := time.Now().Add(time.Duration(ChainSyncRunDuration) * time.Second)
	for time.Now().Before(timeLimit) && chainSync.LastHeight < chainSync.MaxHeightStored {
		err := chainSync.processNextBlock()
		if err != nil {
			logrus.Debugf("FAILURE @%d: %s", chainSync.LastHeight, err.Error())
		}
		time.Sleep(time.Duration(ChainSyncDelay) * time.Millisecond)
	}
	doneChainSyncJob(job)
}

type chainSyncStatus struct {
	JobStatus       *model.JobStatus   `json:"-"`
	Block           *model.Block       `json:"-"`
	Tx              *model.Transaction `json:"-"`
	Vin             *model.Input       `json:"-"`
	Vout            *model.Output      `json:"-"`
	LastHeight      int64              `json:"last_height"`
	MaxHeightStored int64              `json:"max_height_stored"`
	Errors          []syncError        `json:"z_errors"`
}

type syncError struct {
	HeightFound []int64 `json:"height_found"`
	Error       string  `json:"error"`
	Area        string  `json:"area"`
}

func (c *chainSyncStatus) processNextBlock() error {
	c.LastHeight = c.LastHeight + 1
	blockHash, err := lbrycrd.LBRYcrdClient.GetBlockHash(c.LastHeight)
	if err != nil {
		return c.recordAndReturnError(c.LastHeight, "lbrycrd-getblockhash", err)
	}
	lbrycrdBlock, err := lbrycrd.GetBlock(blockHash.String())
	if err != nil {
		return c.recordAndReturnError(c.LastHeight, "mysql-getblock", err)
	}
	recordedBlock, err := model.Blocks(model.BlockWhere.Hash.EQ(blockHash.String())).OneG()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logrus.Warningf("Missing block %d, populating it now", c.LastHeight)
			_, err = processing.ProcessBlock(uint64(c.LastHeight), nil, lbrycrdBlock)
			if err != nil {
				return c.recordAndReturnError(c.LastHeight, "daemon-process-block", err)
			}
		}
		return c.recordAndReturnError(c.LastHeight, "mysql-getblock", err)
	}
	c.Block = recordedBlock
	if err := c.alignBlock(lbrycrdBlock); err != nil {
		return c.recordAndReturnError(c.LastHeight, "block-alignment", err)
	}
	if err := c.alignTxs(recordedBlock, lbrycrdBlock.Tx); err != nil {
		return c.recordAndReturnError(c.LastHeight, "tx-alignment", err)
	}
	return nil
}

func (c *chainSyncStatus) alignTxs(block *model.Block, txHashes []string) error {
	for _, txHash := range txHashes {
		lbrycrdTx, err := lbrycrd.GetRawTransactionResponse(txHash)
		if err != nil {
			return c.recordAndReturnError(c.LastHeight, "tx-hash-creation", err)
		}
		w := model.TransactionWhere
		recordedTx, err := model.Transactions(w.BlockHashID.EQ(null.StringFrom(block.Hash)), w.Hash.EQ(txHash)).OneG()
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				err = processing.ProcessTx(lbrycrdTx, block.BlockTime, uint64(c.LastHeight))
				if err != nil {
					c.recordError(c.LastHeight, "tx-processing", err)
					continue
				}
			}
			return c.recordAndReturnError(c.LastHeight, "mysql-tx", err)
		}
		c.Tx = recordedTx
		if err := c.alignTx(lbrycrdTx); err != nil {
			return c.recordAndReturnError(c.LastHeight, "tx-alignment", err)
		}
		if err := c.alignVins(lbrycrdTx.Vin); err != nil {
			return c.recordAndReturnError(c.LastHeight, "vin-alignment", err)
		}
	}
	return nil
}

func (c chainSyncStatus) alignVouts(vouts []lbrycrd.Vout) error {
	for i, vout := range vouts {
		output := datastore.GetOutput(c.Tx.Hash, uint(vout.N))
		if output == nil {
			err := processing.ProcessVout(&vout, c.Tx, nil, uint64(i))
			if err != nil {
				return errors.Err(err)
			}
		} else {
			c.Vout = output
			err := c.alignVout(vout)
			if err != nil {
				c.recordError(c.LastHeight, "vout-alignment", err)
			}
		}
	}

	return nil
}

func (c *chainSyncStatus) alignVout(v lbrycrd.Vout) error {
	colsToUpdate := make([]string, 0)
	if c.Vout.Value.Float64 != v.Value {
		c.Vout.Value.SetValid(v.Value)
		colsToUpdate = append(colsToUpdate, model.OutputColumns.Value)
	}
	if len(colsToUpdate) > 0 {
		logrus.Debugf("found unaligned vout @%d and Tx %s with the following columns out of alignment: %s", c.LastHeight, c.Tx.Hash, strings.Join(colsToUpdate, ","))
		err := c.Vout.UpdateG(boil.Whitelist(colsToUpdate...))
		if err != nil {
			return errors.Err(err)
		}
	}
	if !c.Vout.ClaimID.IsZero() && !c.Vout.IsSpent {
		return c.alignClaim()
	}
	return nil
}

func (c *chainSyncStatus) alignClaim() error {
	storedClaim := datastore.GetClaim(c.Vout.ClaimID.String)
	if storedClaim == nil {
		return errors.Err("could not find claim with id %s", c.Vout.ClaimID.String)
	}
	helper, err := claim.DecodeClaimHex(storedClaim.ValueAsHex, global.BlockChainName)
	if err != nil {
		return err
	}
	if helper == nil {
		return errors.Err("could not create help for claim %s from ValueAsHex", c.Vout.ClaimID)
	}
	original := *storedClaim
	colsToUpdate := make([]string, 0)
	err = processing.UpdateClaimData(helper, storedClaim)
	if err != nil {
		return err
	}

	//Check for deltas here to update for
	if original.License.String != storedClaim.License.String {
		colsToUpdate = append(colsToUpdate, model.ClaimColumns.License)
	}

	//Update Claim
	if len(colsToUpdate) > 0 {
		logrus.Debugf("found unaligned claim @%d and Tx %s with the following columns out of alignment: %s", c.Vout.ClaimID.String, c.Tx.Hash, strings.Join(colsToUpdate, ","))
		err := storedClaim.UpdateG(boil.Whitelist(colsToUpdate...))
		if err != nil {
			return errors.Err(err)
		}
	}

	return nil
}

func (c *chainSyncStatus) alignVins(vins []lbrycrd.Vin) error {
	for i, vin := range vins {
		input := datastore.GetInput(c.Tx.Hash, false, vin.TxID, uint(vin.Vout))
		if input == nil && len(vin.Coinbase) > 0 {
			input = datastore.GetInput(c.Tx.Hash, true, vin.TxID, uint(vin.Vout))
		}
		if input == nil {
			err := processing.ProcessVin(&vin, c.Tx, nil, uint64(i))
			if err != nil {
				return errors.Err(err)
			}
		} else {
			c.Vin = input
			err := c.alignVin(vin)
			if err != nil {
				c.recordError(c.LastHeight, "vin-alignment", err)
			}
		}
	}
	return nil
}

func (c *chainSyncStatus) alignVin(v lbrycrd.Vin) error {
	colsToUpdate := make([]string, 0)
	if c.Vin.Coinbase.String != v.Coinbase {
		c.Vin.Coinbase.String = v.Coinbase
		c.Vin.Coinbase.Valid = v.Coinbase != ""
		colsToUpdate = append(colsToUpdate, model.InputColumns.Coinbase)
	}
	if c.Vin.Witness.String != strings.Join(v.Witness, ",") {
		c.Vin.Witness.String = strings.Join(v.Witness, ",")
		c.Vin.Witness.Valid = strings.Join(v.Witness, ",") != ""
		colsToUpdate = append(colsToUpdate, model.InputColumns.Witness)
	}
	if v.ScriptSig != nil {
		if c.Vin.ScriptSigHex.String != v.ScriptSig.Hex {
			c.Vin.ScriptSigHex.String = v.ScriptSig.Hex
			c.Vin.ScriptSigHex.Valid = v.ScriptSig.Hex != ""
			colsToUpdate = append(colsToUpdate, model.InputColumns.ScriptSigHex)
		}
		if c.Vin.ScriptSigAsm.String != v.ScriptSig.Asm {
			c.Vin.ScriptSigAsm.String = v.ScriptSig.Asm
			c.Vin.ScriptSigAsm.Valid = v.ScriptSig.Asm != ""
			colsToUpdate = append(colsToUpdate, model.InputColumns.ScriptSigAsm)
		}
	} else {
		if c.Vin.ScriptSigHex.Valid {
			c.Vin.ScriptSigHex.Valid = false
			colsToUpdate = append(colsToUpdate, model.InputColumns.ScriptSigHex)
		}
		if c.Vin.ScriptSigAsm.Valid {
			c.Vin.ScriptSigAsm.Valid = false
			colsToUpdate = append(colsToUpdate, model.InputColumns.ScriptSigAsm)
		}
	}
	srcOutput := datastore.GetOutput(c.Vin.PrevoutHash.String, c.Vin.PrevoutN.Uint)
	if srcOutput != nil {
		if c.Vin.Value.Float64 != srcOutput.Value.Float64 {
			c.Vin.Value.SetValid(srcOutput.Value.Float64)
			colsToUpdate = append(colsToUpdate, model.InputColumns.Value)
		}
	}
	if len(colsToUpdate) > 0 {
		logrus.Debugf("found unaligned vin @%d and Tx %s with the following columns out of alignment: %s", c.LastHeight, c.Tx.Hash, strings.Join(colsToUpdate, ","))
		err := c.Vin.UpdateG(boil.Whitelist(colsToUpdate...))
		if err != nil {
			return errors.Err(err)
		}
	}
	return nil
}

func (c *chainSyncStatus) alignTx(l *lbrycrd.TxRawResult) error {
	colsToUpdate := make([]string, 0)
	if c.Tx.Version != int(l.Version) {
		c.Tx.Version = int(l.Version)
		colsToUpdate = append(colsToUpdate, model.TransactionColumns.Version)
	}
	if c.Tx.TransactionTime.Uint64 != uint64(l.Time) {
		c.Tx.TransactionTime.Uint64 = uint64(l.Time)
		colsToUpdate = append(colsToUpdate, model.TransactionColumns.TransactionTime)
	}
	if c.Tx.TransactionSize != uint64(l.Size) {
		c.Tx.TransactionSize = uint64(l.Size)
		colsToUpdate = append(colsToUpdate, model.TransactionColumns.TransactionSize)
	}
	if c.Tx.LockTime != uint(l.LockTime) {
		c.Tx.LockTime = uint(l.LockTime)
		colsToUpdate = append(colsToUpdate, model.TransactionColumns.LockTime)
	}
	if c.Tx.InputCount != uint(len(l.Vin)) {
		c.Tx.InputCount = uint(len(l.Vin))
		colsToUpdate = append(colsToUpdate, model.TransactionColumns.InputCount)
	}
	if c.Tx.OutputCount != uint(len(l.Vout)) {
		c.Tx.OutputCount = uint(len(l.Vout))
		colsToUpdate = append(colsToUpdate, model.TransactionColumns.OutputCount)
	}
	if !c.Tx.Raw.IsZero() {
		c.Tx.Raw.String = ""
		c.Tx.Raw.Valid = false
		colsToUpdate = append(colsToUpdate, model.TransactionColumns.Raw)
	}
	if len(colsToUpdate) > 0 {
		logrus.Debugf("found unaligned tx @%d and hash %s with the following columns out of alignment: %s", c.LastHeight, c.Tx.Hash, strings.Join(colsToUpdate, ","))
		err := c.Tx.UpdateG(boil.Whitelist(colsToUpdate...))
		if err != nil {
			return errors.Err(err)
		}
	}
	return nil

}

func (c *chainSyncStatus) alignBlock(l *lbrycrd.GetBlockResponse) error {
	colsToUpdate := make([]string, 0)
	if c.Block.Hash != l.Hash {
		c.Block.Hash = l.Hash
		colsToUpdate = append(colsToUpdate, model.BlockColumns.Hash)
	}
	if c.Block.BlockTime != uint64(l.Time) {
		c.Block.BlockTime = uint64(l.Time)
		colsToUpdate = append(colsToUpdate, model.BlockColumns.BlockTime)
	}
	if c.Block.Version != uint64(l.Version) {
		c.Block.Version = uint64(l.Version)
		colsToUpdate = append(colsToUpdate, model.BlockColumns.Version)
	}
	if c.Block.Bits != l.Bits {
		c.Block.Bits = l.Bits
		colsToUpdate = append(colsToUpdate, model.BlockColumns.Bits)
	}
	if c.Block.BlockSize != uint64(l.Size) {
		c.Block.BlockSize = uint64(l.Size)
		colsToUpdate = append(colsToUpdate, model.BlockColumns.BlockSize)
	}
	if c.Block.Chainwork != l.ChainWork {
		c.Block.Chainwork = l.ChainWork
		colsToUpdate = append(colsToUpdate, model.BlockColumns.Chainwork)
	}
	lDifficulty, _ := strconv.ParseFloat(fmt.Sprintf("%.6f", l.Difficulty), 64)
	rDifficulty, _ := strconv.ParseFloat(fmt.Sprintf("%.6f", c.Block.Difficulty), 64)
	if rDifficulty != lDifficulty {
		c.Block.Difficulty = lDifficulty
		colsToUpdate = append(colsToUpdate, model.BlockColumns.Difficulty)
	}
	if c.Block.MerkleRoot != l.MerkleRoot {
		c.Block.MerkleRoot = l.MerkleRoot
		colsToUpdate = append(colsToUpdate, model.BlockColumns.MerkleRoot)
	}
	if c.Block.NameClaimRoot != l.NameClaimRoot {
		c.Block.NameClaimRoot = l.NameClaimRoot
		colsToUpdate = append(colsToUpdate, model.BlockColumns.NameClaimRoot)
	}
	if c.Block.PreviousBlockHash.String != l.PreviousBlockHash {
		c.Block.PreviousBlockHash.SetValid(l.PreviousBlockHash)
		colsToUpdate = append(colsToUpdate, model.BlockColumns.PreviousBlockHash)
	}
	if c.Block.TransactionHashes.String != strings.Join(l.Tx, ",") {
		c.Block.TransactionHashes.SetValid(strings.Join(l.Tx, ","))
		colsToUpdate = append(colsToUpdate, model.BlockColumns.TransactionHashes)
	}
	if c.Block.Nonce != l.Nonce {
		c.Block.Nonce = l.Nonce
		colsToUpdate = append(colsToUpdate, model.BlockColumns.Nonce)
	}
	if c.Block.VersionHex != l.VersionHex {
		c.Block.VersionHex = l.VersionHex
		colsToUpdate = append(colsToUpdate, model.BlockColumns.VersionHex)
	}
	if len(colsToUpdate) > 0 {
		logrus.Debugf("found unaligned block @%d with the following columns out of alignment: %s", c.LastHeight, strings.Join(colsToUpdate, ","))
		err := c.Block.UpdateG(boil.Whitelist(colsToUpdate...))
		if err != nil {
			return errors.Err(err)
		}
	}
	return nil
}

func (c *chainSyncStatus) recordAndReturnError(height int64, area string, err error) error {
	for _, e := range c.Errors {
		if area == e.Area && e.Error == err.Error() {
			e.HeightFound = append(e.HeightFound, height)
		}
	}
	c.Errors = append(c.Errors, syncError{
		HeightFound: []int64{height},
		Error:       err.Error(),
		Area:        area,
	})

	return err
}

func (c *chainSyncStatus) recordError(height int64, area string, err error) {
	_ = c.recordAndReturnError(height, area, err)
}

func (c *chainSyncStatus) updateMaxHeightStored() error {
	lastBlock, err := model.Blocks(qm.OrderBy(model.BlockColumns.Height+" DESC"), qm.Limit(1)).OneG()
	if err != nil {
		return err
	}
	if lastBlock != nil {
		if c.LastHeight >= int64(lastBlock.Height) {
			c.LastHeight = 0 //Reset
			c.Errors = make([]syncError, 0)
		} else {
			c.MaxHeightStored = int64(lastBlock.Height)
		}
	}
	return nil
}

func getChainSyncJobStatus() (*model.JobStatus, error) {
	jobStatus, err := model.FindJobStatusG(chainSyncJob)
	if errors.Is(sql.ErrNoRows, err) {
		syncState := chainSyncStatus{LastHeight: 0}
		bytes, err := json.Marshal(syncState)
		if err != nil {
			return nil, errors.Err(err)
		}
		jobStatus = &model.JobStatus{JobName: chainSyncJob, LastSync: time.Time{}, State: null.JSONFrom(bytes)}
		if err := jobStatus.InsertG(boil.Infer()); err != nil {
			logrus.Panic("Cannot Retrieve/Create JobStatus for " + chainSyncJob)
		}
	} else if err != nil {
		return nil, errors.Err(err)
	}

	err = json.Unmarshal(jobStatus.State.JSON, chainSync)
	if err != nil {
		return nil, errors.Err(err)
	}
	if chainSync.MaxHeightStored == 0 {
		return jobStatus, chainSync.updateMaxHeightStored()
	}

	return jobStatus, nil
}

func doneChainSyncJob(jobStatus *model.JobStatus) {
	jobStatus.LastSync = time.Now()
	jobStatus.IsSuccess = true
	bytes, err := json.Marshal(&chainSync)
	if err != nil {
		logrus.Error(err)
		return
	}
	jobStatus.State.SetValid(bytes)
	if err := jobStatus.UpdateG(boil.Infer()); err != nil {
		logrus.Panic(err)
	}
}
