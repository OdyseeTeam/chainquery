package jobs

import (
	"encoding/json"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"

	"github.com/lbryio/lbry.go/errors"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

const claimTrieSyncJob = "claimtriesyncjob"
const debugClaimTrieSync = true

var expirationHardForkHeight uint = 400155    // https://github.com/lbryio/lbrycrd/pull/137
var hardForkBlocksToExpiration uint = 2102400 // https://github.com/lbryio/lbrycrd/pull/137
var blockHeight uint64
var blocksToExpiration uint = 262974 //Hardcoded! https://lbry.io/faq/claimtrie-implementation
// ClaimTrieSyncRunning is a variable used to show whether or not the job is running already.
var claimTrieSyncRunning = false

var lastSync *claimTrieSyncStatus

type claimTrieSyncStatus struct {
	JobStatus        *model.JobStatus `json:"-"`
	PreviousSyncTime time.Time        `json:"previous_sync"`
	LastHeight       int64            `json:"last_height"`
}

// ClaimTrieSync synchronizes claimtrie information that is calculated and enforced by lbrycrd.
func ClaimTrieSync() {
	if !claimTrieSyncRunning {
		claimTrieSyncRunning = true
		//Run in background so the application can shutdown properly.
		go claimTrieSync()
	}
}

func claimTrieSync() {
	//defer util.TimeTrack(time.Now(), "ClaimTrieSync", "always")
	printDebug("ClaimTrieSync: started... ")
	if lastSync == nil {
		lastSync = &claimTrieSyncStatus{}
	}
	jobStatus, err := getClaimTrieSyncJobStatus()
	if err != nil {
		logrus.Error(err)
	}
	printDebug("ClaimTrieSync: updating spent claims")
	//For Updating claims that are spent ( no longer in claimtrie )
	if err := updateSpentClaims(); err != nil {
		logrus.Error("ClaimTrieSync:", err)
		saveJobError(jobStatus, err)
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
		panic(err)
	}
	printDebug("ClaimTrieSync: Claims to update " + strconv.Itoa(len(updatedClaims)))

	//For syncing the claims
	if err := syncClaims(updatedClaims); err != nil {
		logrus.Error("ClaimTrieSync:", err)
		saveJobError(jobStatus, err)
	}

	//For Setting Controlling Claims
	if err := setControllingClaimForNames(updatedClaims); err != nil {
		logrus.Error("ClaimTrieSync:", err)
		saveJobError(jobStatus, err)
	}

	jobStatus.LastSync = started
	jobStatus.IsSuccess = true
	bytes, err := json.Marshal(&lastSync)
	if err != nil {
		logrus.Error(err)
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

func initControllingWorkers(nrWorkers int, jobs <-chan string, wg *sync.WaitGroup) {

	for i := 0; i < nrWorkers; i++ {
		wg.Add(1)
		go controllingProcessor(jobs, wg)
	}
}

func syncProcessor(jobs <-chan lbrycrd.Claim, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		syncClaim(&job)
	}
}

func controllingProcessor(names <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	for name := range names {
		setBidStateOfClaimsForName(name)
	}
}

func setControllingClaimForNames(claims model.ClaimSlice) error {
	printDebug("ClaimTrieSync: controlling claim status update started... ")
	controlwg := sync.WaitGroup{}
	names := make(map[string]string)
	printDebug("ClaimTrieSync: Making name map...")
	for _, claim := range claims {
		names[claim.Name] = claim.Name
	}
	printDebug("ClaimTrieSync: Finished making name map...")
	setControllingQueue := make(chan string, 100)
	initControllingWorkers(runtime.NumCPU()-1, setControllingQueue, &controlwg)
	for _, name := range names {
		setControllingQueue <- name
	}
	close(setControllingQueue)
	controlwg.Wait()
	printDebug("ClaimTrieSync: controlling claim status update complete... ")

	return nil
}

func setBidStateOfClaimsForName(name string) {
	claims, _ := model.Claims(
		qm.Where(model.ClaimColumns.Name+"=?", name),
		qm.Where(model.ClaimColumns.BidState+"!=?", "Spent"),
		qm.Where(model.ClaimColumns.ValidAtHeight+"<=?", blockHeight),
		qm.OrderBy(model.ClaimColumns.EffectiveAmount+" DESC")).AllG()

	foundControlling := false
	for _, claim := range claims {
		if !foundControlling && getClaimStatus(claim) == "Active" {
			if claim.BidState != "Controlling" {
				claim.BidState = "Controlling"
				err := datastore.PutClaim(claim)
				if err != nil {
					panic(err)
				}
			}
			foundControlling = true
		} else {
			status := getClaimStatus(claim)
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

func syncClaims(claims model.ClaimSlice) error {

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

func getClaimStatus(claim *model.Claim) string {
	status := "Accepted"
	//Transaction and output should never be missing if the claim exists.
	transaction, err := claim.TransactionHash().OneG()
	if err != nil {
		logrus.Error("could not find transaction ", claim.TransactionHashID, " : ", err)
		return status
	}

	output, err := transaction.Outputs(qm.Where(model.OutputColumns.Vout+"=?", claim.Vout)).OneG()
	if err != nil {
		logrus.Error("could not find output ", claim.TransactionHashID, "-", claim.Vout, " : ", err)
		return status
	}

	if output.IsSpent {
		status = "Spent" //Should be unreachable because claim would be out of claimtrie if spent.
	}
	height := claim.Height
	if GetIsExpiredAtHeight(height, uint(blockHeight)) {
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

func getSpentClaimsToUpdate() (model.ClaimSlice, error) {

	claim := model.TableNames.Claim

	claimID := claim + "." + model.ClaimColumns.ID
	claimClaimID := claim + "." + model.ClaimColumns.ClaimID
	claimTxByHash := claim + "." + model.ClaimColumns.TransactionHashID
	claimVout := claim + "." + model.ClaimColumns.Vout
	claimBidState := claim + "." + model.ClaimColumns.BidState

	output := model.TableNames.Output
	outputTxHash := output + "." + model.OutputColumns.TransactionHash
	outputVout := output + "." + model.OutputColumns.Vout
	outputClaimID := output + "." + model.OutputColumns.ClaimID
	outputIsSpent := output + "." + model.OutputColumns.IsSpent
	outputModifiedAt := output + "." + model.OutputColumns.ModifiedAt

	query := `
		SELECT ` + claimClaimID + `,` + claimID + ` 
		FROM ` + output + `
		INNER JOIN ` + claim + ` ON ` + claimID + ` = ` + outputClaimID + ` 
			AND ` + claimTxByHash + ` = ` + outputTxHash + ` 
			AND ` + claimVout + ` = ` + outputVout + `
		WHERE ` + outputModifiedAt + ` > ? 
		AND ` + outputIsSpent + ` = ? 
		AND ` + claimBidState + ` != ?`
	printDebug(query)
	return model.Claims(qm.SQL(query, lastSync.PreviousSyncTime, 1, "Spent")).AllG()
}

func updateSpentClaims() error {
	claims, err := getSpentClaimsToUpdate()
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
	if err != nil {
		return nil, errors.Err(err)
	}
	if jobStatus == nil {
		jobStatus = &model.JobStatus{JobName: claimTrieSyncJob, LastSync: time.Time{}}
		if err := jobStatus.InsertG(boil.Infer()); err != nil {
			logrus.Panic("Cannot Retrieve/Create JobStatus for " + claimTrieSyncJob)
		}
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
