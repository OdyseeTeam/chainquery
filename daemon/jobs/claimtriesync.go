package jobs

import (
	"database/sql"
	"encoding/json"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"

	"github.com/lbryio/lbry.go/extras/errors"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

const claimTrieSyncJob = "claimtriesyncjob"
const debugClaimTrieSync = false

var expirationHardForkHeight uint = 400155    // https://github.com/lbryio/lbrycrd/pull/137
var hardForkBlocksToExpiration uint = 2102400 // https://github.com/lbryio/lbrycrd/pull/137
var blockHeight uint64
var blocksToExpiration uint = 262974 //Hardcoded! https://lbry.com/faq/claimtrie-implementation
// ClaimTrieSyncRunning is a variable used to show whether or not the job is running already.
var claimTrieSyncRunning = false

var lastSync *claimTrieSyncStatus

type claimTrieSyncStatus struct {
	JobStatus        *model.JobStatus `json:"-"`
	PreviousSyncTime time.Time        `json:"previous_sync"`
	LastHeight       int64            `json:"last_height"`
}

// ClaimTrieSyncAsync synchronizes claimtrie information that is calculated and enforced by lbrycrd.
func ClaimTrieSyncAsync() {
	if !claimTrieSyncRunning {
		claimTrieSyncRunning = true
		//Run in background so the application can shutdown properly.
		go ClaimTrieSync()
	}
}

// ClaimTrieSync syncs the claim trie bidstate, effective amount and effective height
func ClaimTrieSync() {
	//defer util.TimeTrack(time.Now(), "ClaimTrieSync", "always")
	printDebug("ClaimTrieSync: started... ")
	if lastSync == nil {
		lastSync = &claimTrieSyncStatus{}
	}
	jobStatus, err := getClaimTrieSyncJobStatus()
	if err != nil {
		logrus.Error(err)
		return
	}
	printDebug("ClaimTrieSync: updating spent claims")
	//For Updating claims that are spent ( no longer in claimtrie )
	if err := updateSpentClaims(); err != nil {
		logrus.Error("ClaimTrieSync:", err)
		saveJobError(jobStatus, err)
		return
	}

	started := time.Now()
	printDebug("ClaimTrieSync: getting block height")
	//Get blockheight for calculating expired status
	count, err := lbrycrd.GetBlockCount()
	if err != nil {
		logrus.Error("ClaimTrieSync: Error getting block height", err)
		return
	}
	blockHeight = *count

	lastSync.PreviousSyncTime = jobStatus.LastSync
	lastSync.LastHeight = int64(blockHeight)

	printDebug("ClaimTrieSync: getting updated claims...")
	updatedClaims, err := getUpdatedClaims(jobStatus)
	if err != nil {
		logrus.Error("ClaimTrieSync:", err)
		saveJobError(jobStatus, err)
		return
	}
	printDebug("ClaimTrieSync: Claims to update " + strconv.Itoa(len(updatedClaims)))

	//For syncing the claims
	if err := SyncClaims(updatedClaims); err != nil {
		logrus.Error("ClaimTrieSync:", err)
		saveJobError(jobStatus, err)
		return
	}

	//For Setting Controlling Claims
	if err := SetControllingClaimForNames(updatedClaims, blockHeight); err != nil {
		logrus.Error("ClaimTrieSync:", err)
		saveJobError(jobStatus, err)
		return
	}

	jobStatus.LastSync = started
	jobStatus.IsSuccess = true
	bytes, err := json.Marshal(&lastSync)
	if err != nil {
		logrus.Error(err)
		return
	}
	jobStatus.State.SetValid(bytes)
	if err := jobStatus.UpdateG(boil.Infer()); err != nil {
		logrus.Panic(err)
	}
	printDebug("ClaimTrieSync: Processed " + strconv.Itoa(len(updatedClaims)) + " claims.")
	claimTrieSyncRunning = false
}

func initSyncWorkers(nrWorkers int, jobs <-chan lbrycrd.Claim, wg *sync.WaitGroup) {

	for i := 0; i < nrWorkers; i++ {
		wg.Add(1)
		go syncProcessor(jobs, wg)
	}
}

func initControllingWorkers(nrWorkers int, jobs <-chan string, wg *sync.WaitGroup, atHeight uint64) {

	for i := 0; i < nrWorkers; i++ {
		wg.Add(1)
		go controllingProcessor(jobs, wg, atHeight)
	}
}

func syncProcessor(jobs <-chan lbrycrd.Claim, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		syncClaim(&job)
	}
}

func controllingProcessor(names <-chan string, wg *sync.WaitGroup, atHeight uint64) {
	defer wg.Done()
	for name := range names {
		setBidStateOfClaimsForName(name, atHeight)
	}
}

// SetControllingClaimForNames sets the bid state for claims with these names.
func SetControllingClaimForNames(claims model.ClaimSlice, atHeight uint64) error {
	printDebug("ClaimTrieSync: controlling claim status update started... ")
	controlwg := sync.WaitGroup{}
	names := make(map[string]string)
	printDebug("ClaimTrieSync: Making name map...")
	for _, claim := range claims {
		names[claim.Name] = claim.Name
	}
	printDebug("ClaimTrieSync: Finished making name map...[", len(names), "]")
	setControllingQueue := make(chan string, 100)
	initControllingWorkers(runtime.NumCPU()-1, setControllingQueue, &controlwg, atHeight)
	for _, name := range names {
		setControllingQueue <- name
	}
	close(setControllingQueue)
	controlwg.Wait()
	printDebug("ClaimTrieSync: controlling claim status update complete... ")

	return nil
}

func setBidStateOfClaimsForName(name string, atHeight uint64) {
	claims, _ := model.Claims(
		qm.Where(model.ClaimColumns.Name+"=?", name),
		qm.Where(model.ClaimColumns.ValidAtHeight+"<=?", atHeight),
		qm.OrderBy(model.ClaimColumns.EffectiveAmount+" DESC")).AllG()
	printDebug("found ", len(claims), " claims matching the name ", name)
	foundControlling := false
	for _, claim := range claims {
		if !foundControlling && getClaimStatus(claim, atHeight) == "Active" {
			if claim.BidState != "Controlling" {
				claim.BidState = "Controlling"
				err := datastore.PutClaim(claim)
				if err != nil {
					panic(err)
				}
			}
			foundControlling = true
		} else {
			status := getClaimStatus(claim, atHeight)
			if status != claim.BidState {
				claim.BidState = status
				err := datastore.PutClaim(claim)
				if err != nil {
					panic(err)
				}
			}
		}
	}
}

// SyncClaims syncs the claims' with these names effective amount and valid at height with the lbrycrd claimtrie.
func SyncClaims(claims model.ClaimSlice) error {

	printDebug("ClaimTrieSync: claim  update started... ")
	syncwg := sync.WaitGroup{}
	processingQueue := make(chan lbrycrd.Claim, 100)
	initSyncWorkers(runtime.NumCPU()-1, processingQueue, &syncwg)
	for i, claim := range claims {
		if i%1000 == 0 {
			printDebug("ClaimTrieSync: syncing ", i, " of ", len(claims), " queued - ", len(processingQueue))
		}
		claims, err := lbrycrd.GetClaimsForName(claim.Name)
		if err != nil {
			logrus.Error("ClaimTrieSync: Could not get claims for name: ", claim.Name, " Error: ", err)
		}
		for _, claimJSON := range claims.Claims {
			processingQueue <- claimJSON
		}
	}
	close(processingQueue)
	syncwg.Wait()

	printDebug("ClaimTrieSync: claim  update complete... ")

	return nil
}

func syncClaim(claimJSON *lbrycrd.Claim) {
	hasChanges := false
	claim := datastore.GetClaim(claimJSON.ClaimID)
	if claim == nil {
		unknown, _ := model.AbnormalClaims(qm.Where(model.AbnormalClaimColumns.ClaimID+"=?", claimJSON.ClaimID)).OneG()
		if unknown == nil {
			printDebug("ClaimTrieSync: Missing Claim ", claimJSON.ClaimID, " ", claimJSON.TxID, " ", claimJSON.N)
		}
		return
	}
	if claim.ValidAtHeight != uint(claimJSON.ValidAtHeight) {
		claim.ValidAtHeight = uint(claimJSON.ValidAtHeight)
		hasChanges = true
	}
	if claim.EffectiveAmount != claimJSON.EffectiveAmount {
		claim.EffectiveAmount = claimJSON.EffectiveAmount
		hasChanges = true
	}
	if hasChanges {
		if err := datastore.PutClaim(claim); err != nil {
			logrus.Error("ClaimTrieSync: unable to sync claim ", claim.ClaimID, ". JSON-", claimJSON)
			printDebug("Error: ", err)
		}
	}
}

func getClaimStatus(claim *model.Claim, atHeight uint64) string {
	status := "Accepted"
	var transaction *model.Transaction
	var err error
	if !claim.TransactionHashUpdate.IsZero() {
		transaction, err = model.Transactions(model.TransactionWhere.Hash.EQ(claim.TransactionHashUpdate.String)).OneG()
	} else { //Transaction and output should never be missing if the claim exists.
		transaction, err = claim.TransactionHash().OneG()
	}

	if err != nil {
		logrus.Error("could not find transaction ", claim.TransactionHashID, " : ", err)
		return status
	}

	output, err := transaction.Outputs(qm.Where(model.OutputColumns.Vout+"=?", claim.Vout)).OneG()
	if err != nil {
		logrus.Error("could not find output ", claim.TransactionHashID, "-", claim.Vout, " : ", err)
		return "ERROR"
	}

	if output.IsSpent {
		status = "Spent"
	}
	height := claim.Height
	if GetIsExpiredAtHeight(height, uint(atHeight)) {
		status = "Expired"
	}

	//Neither Spent or Expired = Active
	if status == "Accepted" {
		status = "Active"
	}

	return status
}

//GetIsExpiredAtHeight checks the claim height compared to the current height to determine expiration.
func GetIsExpiredAtHeight(height, blockHeight uint) bool {
	if height == 0 {
		return false
	}
	if height >= expirationHardForkHeight {
		// https://github.com/lbryio/lbrycrd/pull/137 - HardFork extends claim expiration.
		if height+hardForkBlocksToExpiration < blockHeight {
			return true
		}
	} else if height+blocksToExpiration >= expirationHardForkHeight {
		// https://github.com/lbryio/lbrycrd/pull/137 - HardFork extends claim expiration.
		if height+hardForkBlocksToExpiration < blockHeight {
			return true
		}
	} else {
		if height+blocksToExpiration < blockHeight {
			return true
		}
	}
	return false
}

func getUpdatedClaims(jobStatus *model.JobStatus) (model.ClaimSlice, error) {
	claimIDCol := model.TableNames.Claim + "." + model.ClaimColumns.ClaimID
	claimNameCol := model.TableNames.Claim + "." + model.ClaimColumns.Name
	supportedIDCol := model.TableNames.Support + "." + model.SupportColumns.SupportedClaimID
	supportModifiedCol := model.TableNames.Support + "." + model.SupportColumns.ModifiedAt
	claimModifiedCol := model.TableNames.Claim + "." + model.ClaimColumns.ModifiedAt
	claimValidAtHeight := model.TableNames.Claim + "." + model.ClaimColumns.ValidAtHeight
	sqlFormat := "2006-01-02 15:04:05"
	lastsync := jobStatus.LastSync.Format(sqlFormat)
	lastSyncStr := "'" + lastsync + "'"
	query := `
		SELECT ` + claimNameCol + `,` + claimIDCol + `
		FROM ` + model.TableNames.Claim + `
		WHERE ` + claimNameCol + ` IN 
		(
			SELECT
				` + claimNameCol + `
			FROM ` + model.TableNames.Claim + ` 
			LEFT JOIN ` + model.TableNames.Support + ` 
				ON ( ` + supportedIDCol + ` = ` + claimIDCol + ` AND ` + supportModifiedCol + ` >= ` + lastSyncStr + ` )
			WHERE ` + claimModifiedCol + ` >= ` + lastSyncStr + ` 
			OR ` + supportedIDCol + ` IS NOT NULL 
			OR ` + claimValidAtHeight + ` >= ? 
			GROUP BY  ` + claimNameCol + `
		)
`
	printDebug(query)
	return model.Claims(qm.SQL(query, lastSync.LastHeight)).AllG()

}

func getSpentClaimsToUpdate(hasUpdate bool) (model.ClaimSlice, error) {

	claim := model.TableNames.Claim

	claimID := claim + "." + model.ClaimColumns.ID
	claimTxByHashUpdate := claim + "." + model.ClaimColumns.TransactionHashUpdate
	claimVoutUpdate := claim + "." + model.ClaimColumns.VoutUpdate
	claimClaimID := claim + "." + model.ClaimColumns.ClaimID
	claimTxByHash := claim + "." + model.ClaimColumns.TransactionHashID
	claimVout := claim + "." + model.ClaimColumns.Vout
	claimBidState := claim + "." + model.ClaimColumns.BidState

	output := model.TableNames.Output
	outputTxHash := output + "." + model.OutputColumns.TransactionHash
	outputVout := output + "." + model.OutputColumns.Vout
	outputIsSpent := output + "." + model.OutputColumns.IsSpent
	outputModifiedAt := output + "." + model.OutputColumns.ModifiedAt

	claimJoin := `INNER JOIN ` + claim + ` ON ` + claimTxByHash + ` = ` + outputTxHash + `
	AND ` + claimVout + ` = ` + outputVout

	if hasUpdate {
		claimJoin = `INNER JOIN ` + claim + ` ON ` + claimTxByHashUpdate + ` = ` + outputTxHash + `
		AND ` + claimVoutUpdate + ` = ` + outputVout
	}

	query := `
		SELECT ` + claimClaimID + `,` + claimID + `,` + claimTxByHashUpdate + ` 
		FROM ` + output + `
		` + claimJoin + `
		WHERE ` + outputModifiedAt + ` >= ? 
		AND ` + outputIsSpent + ` = ? 
		AND ` + claimBidState + ` != ?`
	printDebug(query)
	return model.Claims(qm.SQL(query, lastSync.PreviousSyncTime, 1, "Spent")).AllG()
}

func updateSpentClaims() error {

	//Claims without updates
	claims, err := getSpentClaimsToUpdate(false)
	if err != nil {
		return err
	}
	for _, claim := range claims {
		if !claim.TransactionHashUpdate.IsZero() {
			continue
		}
		claim.BidState = "Spent"
		claim.ModifiedAt = time.Now()
		if err := claim.UpdateG(boil.Whitelist(model.ClaimColumns.BidState, model.ClaimColumns.ModifiedAt)); err != nil {
			return err
		}
	}
	//Claims without updates
	claims, err = getSpentClaimsToUpdate(true)
	if err != nil {
		return err
	}
	for _, claim := range claims {
		claim.BidState = "Spent"
		claim.ModifiedAt = time.Now()
		if err := claim.UpdateG(boil.Whitelist(model.ClaimColumns.BidState, model.ClaimColumns.ModifiedAt)); err != nil {
			return err
		}
	}
	return nil
}

func getClaimTrieSyncJobStatus() (*model.JobStatus, error) {
	jobStatus, err := model.FindJobStatusG(claimTrieSyncJob)
	if errors.Is(sql.ErrNoRows, err) {
		syncState := claimTrieSyncStatus{PreviousSyncTime: time.Unix(458265600, 0), LastHeight: 0}
		bytes, err := json.Marshal(syncState)
		if err != nil {
			return nil, errors.Err(err)
		}
		jobStatus = &model.JobStatus{JobName: claimTrieSyncJob, LastSync: time.Time{}, State: null.JSONFrom(bytes)}
		if err := jobStatus.InsertG(boil.Infer()); err != nil {
			logrus.Panic("Cannot Retrieve/Create JobStatus for " + claimTrieSyncJob)
		}
	} else if err != nil {
		return nil, errors.Err(err)
	}

	err = json.Unmarshal(jobStatus.State.JSON, lastSync)
	if err != nil {
		return nil, errors.Err(err)
	}

	return jobStatus, nil
}

func saveJobError(jobStatus *model.JobStatus, error error) {
	jobStatus.ErrorMessage.SetValid(error.Error())
	jobStatus.IsSuccess = false
	if err := jobStatus.UpsertG(boil.Infer(), boil.Infer()); err != nil {
		logrus.Error(errors.Prefix("Saving Job Error Message "+error.Error(), err))
	}
}

func printDebug(args ...interface{}) {
	if debugClaimTrieSync {
		logrus.Info(args...)
	} else {
		logrus.Debug(args...)
	}
}
