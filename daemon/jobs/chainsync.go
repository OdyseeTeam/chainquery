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
	JobStatus       *model.JobStatus `json:"-"`
	LastHeight      int64            `json:"last_height"`
	MaxHeightStored int64            `json:"max_height_stored"`
	Errors          []syncError      `json:"errors"`
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
		return c.recordError(c.LastHeight, "lbrycrd-getblockhash", err)
	}
	lbrycrdBlock, err := lbrycrd.GetBlock(blockHash.String())
	if err != nil {
		return c.recordError(c.LastHeight, "mysql-getblock", err)
	}
	recordedBlock, err := model.Blocks(model.BlockWhere.Hash.EQ(blockHash.String())).OneG()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logrus.Warningf("Missing block %d, populating it now", c.LastHeight)
			_, err = processing.ProcessBlock(uint64(c.LastHeight), nil, lbrycrdBlock)
			if err != nil {
				return c.recordError(c.LastHeight, "daemon-process-block", err)
			}
		}
		return c.recordError(c.LastHeight, "mysql-getblock", err)
	}
	if err := c.alignBlocks(recordedBlock, lbrycrdBlock); err != nil {
		return c.recordError(c.LastHeight, "block-alignment", err)
	}
	return nil
}

func (c *chainSyncStatus) alignBlocks(r *model.Block, l *lbrycrd.GetBlockResponse) error {
	colsToUpdate := make([]string, 0)
	if r.Hash != l.Hash {
		r.Hash = l.Hash
		colsToUpdate = append(colsToUpdate, model.BlockColumns.Hash)
	}
	if r.BlockTime != uint64(l.Time) {
		r.BlockTime = uint64(l.Time)
		colsToUpdate = append(colsToUpdate, model.BlockColumns.BlockTime)
	}
	if r.Version != uint64(l.Version) {
		r.Version = uint64(l.Version)
		colsToUpdate = append(colsToUpdate, model.BlockColumns.Version)
	}
	if r.Bits != l.Bits {
		r.Bits = l.Bits
		colsToUpdate = append(colsToUpdate, model.BlockColumns.Bits)
	}
	if r.BlockSize != uint64(l.Size) {
		r.BlockSize = uint64(l.Size)
		colsToUpdate = append(colsToUpdate, model.BlockColumns.BlockSize)
	}
	if r.Chainwork != l.ChainWork {
		r.Chainwork = l.ChainWork
		colsToUpdate = append(colsToUpdate, model.BlockColumns.Chainwork)
	}
	difficultyPrecision := 8 //MySQL DOUBLE(50,8)
	if util.ToFixed(r.Difficulty, difficultyPrecision) != util.ToFixed(l.Difficulty, difficultyPrecision) {
		r.Difficulty = util.ToFixed(l.Difficulty, difficultyPrecision)
		colsToUpdate = append(colsToUpdate, model.BlockColumns.Difficulty)
	}
	if r.MerkleRoot != l.MerkleRoot {
		r.MerkleRoot = l.MerkleRoot
		colsToUpdate = append(colsToUpdate, model.BlockColumns.MerkleRoot)
	}
	if r.NameClaimRoot != l.NameClaimRoot {
		r.NameClaimRoot = l.NameClaimRoot
		colsToUpdate = append(colsToUpdate, model.BlockColumns.NameClaimRoot)
	}
	if r.PreviousBlockHash.String != l.PreviousBlockHash {
		r.PreviousBlockHash.SetValid(l.PreviousBlockHash)
		colsToUpdate = append(colsToUpdate, model.BlockColumns.PreviousBlockHash)
	}
	if r.TransactionHashes.String != strings.Join(l.Tx, ",") {
		r.TransactionHashes.SetValid(strings.Join(l.Tx, ","))
		colsToUpdate = append(colsToUpdate, model.BlockColumns.TransactionHashes)
	}
	if r.Nonce != l.Nonce {
		r.Nonce = l.Nonce
		colsToUpdate = append(colsToUpdate, model.BlockColumns.Nonce)
	}
	if r.VersionHex != l.VersionHex {
		r.VersionHex = l.VersionHex
		colsToUpdate = append(colsToUpdate, model.BlockColumns.VersionHex)
	}
	if len(colsToUpdate) > 0 {
		logrus.Debugf("found unaligned block @%d with the following columns out of alignment: %s", c.LastHeight, strings.Join(colsToUpdate, ","))
		err := r.UpdateG(boil.Whitelist(colsToUpdate...))
		if err != nil {
			return errors.Err(err)
		}
	}
	return nil
}

func (c *chainSyncStatus) recordError(height int64, area string, err error) error {
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
