package jobs

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/lbryio/chainquery/daemon/processing"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/chainquery/util"

	"github.com/lbryio/lbry.go/extras/errors"

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
	RecordedBlock   *model.Block       `json:"-"`
	RecordedTx      *model.Transaction `json:"-"`
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
	c.RecordedBlock = recordedBlock
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
		c.RecordedTx = recordedTx
		if err := c.alignTx(lbrycrdTx); err != nil {
			return c.recordAndReturnError(c.LastHeight, "tx-alignment", err)
		}
	}
	return nil
}

func (c *chainSyncStatus) alignTx(l *lbrycrd.TxRawResult) error {
	colsToUpdate := make([]string, 0)
	if c.RecordedTx.Version != int(l.Version) {
		c.RecordedTx.Version = int(l.Version)
		colsToUpdate = append(colsToUpdate, model.TransactionColumns.Version)
	}
	if c.RecordedTx.TransactionTime.Uint64 != uint64(l.Time) {
		c.RecordedTx.TransactionTime.Uint64 = uint64(l.Time)
		colsToUpdate = append(colsToUpdate, model.TransactionColumns.TransactionTime)
	}
	if c.RecordedTx.TransactionSize != uint64(l.Size) {
		c.RecordedTx.TransactionSize = uint64(l.Size)
		colsToUpdate = append(colsToUpdate, model.TransactionColumns.TransactionSize)
	}
	if c.RecordedTx.LockTime != uint(l.LockTime) {
		c.RecordedTx.LockTime = uint(l.LockTime)
		colsToUpdate = append(colsToUpdate, model.TransactionColumns.LockTime)
	}
	if c.RecordedTx.InputCount != uint(len(l.Vin)) {
		c.RecordedTx.InputCount = uint(len(l.Vin))
		colsToUpdate = append(colsToUpdate, model.TransactionColumns.InputCount)
	}
	if c.RecordedTx.OutputCount != uint(len(l.Vout)) {
		c.RecordedTx.OutputCount = uint(len(l.Vout))
		colsToUpdate = append(colsToUpdate, model.TransactionColumns.OutputCount)
	}
	if c.RecordedTx.Raw.String != l.Hex {
		c.RecordedTx.Raw.SetValid(l.Hex)
		colsToUpdate = append(colsToUpdate, model.TransactionColumns.Raw)
	}
	if len(colsToUpdate) > 0 {
		logrus.Debugf("found unaligned tx @%d and hash %s with the following columns out of alignment: %s", c.LastHeight, c.RecordedTx.Hash, strings.Join(colsToUpdate, ","))
		err := c.RecordedTx.UpdateG(boil.Whitelist(colsToUpdate...))
		if err != nil {
			return errors.Err(err)
		}
	}
	return nil

}

func (c *chainSyncStatus) alignBlock(l *lbrycrd.GetBlockResponse) error {
	colsToUpdate := make([]string, 0)
	if c.RecordedBlock.Hash != l.Hash {
		c.RecordedBlock.Hash = l.Hash
		colsToUpdate = append(colsToUpdate, model.BlockColumns.Hash)
	}
	if c.RecordedBlock.BlockTime != uint64(l.Time) {
		c.RecordedBlock.BlockTime = uint64(l.Time)
		colsToUpdate = append(colsToUpdate, model.BlockColumns.BlockTime)
	}
	if c.RecordedBlock.Version != uint64(l.Version) {
		c.RecordedBlock.Version = uint64(l.Version)
		colsToUpdate = append(colsToUpdate, model.BlockColumns.Version)
	}
	if c.RecordedBlock.Bits != l.Bits {
		c.RecordedBlock.Bits = l.Bits
		colsToUpdate = append(colsToUpdate, model.BlockColumns.Bits)
	}
	if c.RecordedBlock.BlockSize != uint64(l.Size) {
		c.RecordedBlock.BlockSize = uint64(l.Size)
		colsToUpdate = append(colsToUpdate, model.BlockColumns.BlockSize)
	}
	if c.RecordedBlock.Chainwork != l.ChainWork {
		c.RecordedBlock.Chainwork = l.ChainWork
		colsToUpdate = append(colsToUpdate, model.BlockColumns.Chainwork)
	}
	difficultyPrecision := 8 //MySQL DOUBLE(50,8)
	if util.ToFixed(c.RecordedBlock.Difficulty, difficultyPrecision) != util.ToFixed(l.Difficulty, difficultyPrecision) {
		c.RecordedBlock.Difficulty = util.ToFixed(l.Difficulty, difficultyPrecision)
		colsToUpdate = append(colsToUpdate, model.BlockColumns.Difficulty)
	}
	if c.RecordedBlock.MerkleRoot != l.MerkleRoot {
		c.RecordedBlock.MerkleRoot = l.MerkleRoot
		colsToUpdate = append(colsToUpdate, model.BlockColumns.MerkleRoot)
	}
	if c.RecordedBlock.NameClaimRoot != l.NameClaimRoot {
		c.RecordedBlock.NameClaimRoot = l.NameClaimRoot
		colsToUpdate = append(colsToUpdate, model.BlockColumns.NameClaimRoot)
	}
	if c.RecordedBlock.PreviousBlockHash.String != l.PreviousBlockHash {
		c.RecordedBlock.PreviousBlockHash.SetValid(l.PreviousBlockHash)
		colsToUpdate = append(colsToUpdate, model.BlockColumns.PreviousBlockHash)
	}
	if c.RecordedBlock.TransactionHashes.String != strings.Join(l.Tx, ",") {
		c.RecordedBlock.TransactionHashes.SetValid(strings.Join(l.Tx, ","))
		colsToUpdate = append(colsToUpdate, model.BlockColumns.TransactionHashes)
	}
	if c.RecordedBlock.Nonce != l.Nonce {
		c.RecordedBlock.Nonce = l.Nonce
		colsToUpdate = append(colsToUpdate, model.BlockColumns.Nonce)
	}
	if c.RecordedBlock.VersionHex != l.VersionHex {
		c.RecordedBlock.VersionHex = l.VersionHex
		colsToUpdate = append(colsToUpdate, model.BlockColumns.VersionHex)
	}
	if len(colsToUpdate) > 0 {
		logrus.Debugf("found unaligned block @%d with the following columns out of alignment: %s", c.LastHeight, strings.Join(colsToUpdate, ","))
		err := c.RecordedBlock.UpdateG(boil.Whitelist(colsToUpdate...))
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
