package jobs

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/metrics"
	"github.com/lbryio/chainquery/model"

	"github.com/lbryio/lbry.go/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/query"

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
	metrics.JobLoad.WithLabelValues("claimtrie_sync").Inc()
	defer metrics.JobLoad.WithLabelValues("claimtrie_sync").Dec()
	defer metrics.Job(time.Now(), "claimtrie_sync")
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
	setControllingQueue := make(chan string, 1000)
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
	printDebug("ClaimTrieSync: found ", len(claims), " claims matching the name ", name)
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
	claimNameMap := make(map[string]bool)
	for _, claim := range claims {
		claimNameMap[claim.Name] = true
	}
	var names []string
	for name := range claimNameMap {
		names = append(names, name)
	}
	printDebug("ClaimTrieSync: ", len(names), " names to sync from lbrycrd...")
	syncwg := sync.WaitGroup{}
	processingQueue := make(chan lbrycrd.Claim, 1000)
	initSyncWorkers(runtime.NumCPU()-1, processingQueue, &syncwg)
	for i, name := range names {
		if i%1000 == 0 {
			printDebug("ClaimTrieSync: syncing ", i, " of ", len(names), " queued - ", len(processingQueue))
		}
		claims, err := lbrycrd.GetClaimsForName(name)
		if err != nil {
			logrus.Error("ClaimTrieSync: Could not get claims for name: ", name, " Error: ", err)
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
	c := model.ClaimColumns
	claim, err := model.Claims(qm.Select(c.ID, c.ValidAtHeight, c.EffectiveAmount), model.ClaimWhere.ClaimID.EQ(claimJSON.ClaimID)).OneG()
	if err == sql.ErrNoRows {
		unknown, _ := model.AbnormalClaims(qm.Where(model.AbnormalClaimColumns.ClaimID+"=?", claimJSON.ClaimID)).OneG()
		if unknown == nil {
			printDebug("ClaimTrieSync: Missing Claim ", claimJSON.ClaimID, " ", claimJSON.TxID, " ", claimJSON.N)
		}
		return
	}
	if err != nil {
		logrus.Error("ClaimTrieSync: ", err)
		return
	}
	if claim.ValidAtHeight != uint(claimJSON.ValidAtHeight) {
		claim.ValidAtHeight = uint(claimJSON.ValidAtHeight)
		hasChanges = true
	}
	if claim.EffectiveAmount != claimJSON.EffectiveAmount {
		if claimJSON.PendingAmount != 0 && claim.EffectiveAmount != claimJSON.PendingAmount {
			claim.EffectiveAmount = claimJSON.EffectiveAmount
		} else {
			claim.EffectiveAmount = claimJSON.EffectiveAmount
		}
		hasChanges = true
	}
	if hasChanges {
		err := claim.UpdateG(boil.Whitelist(c.ValidAtHeight, c.EffectiveAmount))
		if err != nil {
			logrus.Error("ClaimTrieSync: unable to sync claim ", claim.ClaimID, ". JSON-", claimJSON)
			printDebug("Error: ", err)
		}
	}
}

func getClaimStatus(claim *model.Claim, atHeight uint64) string {
	status := "Accepted"
	var transaction *model.Transaction
	var err error
	t := model.TransactionColumns
	if !claim.TransactionHashUpdate.IsZero() {
		transaction, err = model.Transactions(qm.Select(t.ID), model.TransactionWhere.Hash.EQ(claim.TransactionHashUpdate.String)).OneG()
	} else { //Transaction and output should never be missing if the claim exists.
		transaction, err = claim.TransactionHash(qm.Select(t.ID)).OneG()
	}

	if err != nil {
		logrus.Error("could not find transaction ", claim.TransactionHashID, " : ", err)
		return status
	}
	o := model.OutputColumns
	output, err := transaction.Outputs(qm.Select(o.ID, o.IsSpent), qm.Where(model.OutputColumns.Vout+"=?", claim.VoutUpdate)).OneG()
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
	prevNamesLength := 0
	// CLAIMS THAT HAVE SUPPORTS THAT WERE MODIFIED [SELECT support.supported_claim_id FROM support WHERE support.modified_at >= '2019-11-03 19:48:58';]
	s := model.SupportColumns
	supports, err := model.Supports(qm.Select(s.SupportedClaimID), model.SupportWhere.ModifiedAt.GTE(jobStatus.LastSync)).AllG()
	if err != nil {
		return nil, errors.Err(err)
	}
	var claimids []interface{}
	for _, support := range supports {
		claimids = append(claimids, support.SupportedClaimID)
	}
	c := model.ClaimColumns
	upTo := 5000
	var claims model.ClaimSlice
	var namesMap map[string]bool
	for len(claimids) > 0 {
		if len(claimids) < upTo {
			upTo = len(claimids)
		}
		var err error
		toFind := claimids[:upTo]
		claims, err = model.Claims(qm.Select("DISTINCT "+c.Name), qm.WhereIn(c.ClaimID+" IN ?", toFind...)).AllG()
		if err != nil {
			return nil, errors.Err(err)
		}
		namesMap = updateNameList(namesMap, claims)
		claimids = claimids[upTo:]
	}

	prevNamesLength = len(namesMap)
	logrus.Debugf("found %d new names from support modifications", prevNamesLength)
	// CLAIMS THAT WERE MODIFIED [SELECT DISTINCT claim.name FROM claim WHERE claim.modified_at >= '2019-11-03 19:48:58';]
	claims, err = model.Claims(qm.Select("DISTINCT "+c.Name), model.ClaimWhere.ModifiedAt.GTE(jobStatus.LastSync)).AllG()
	if err != nil {
		return nil, errors.Err(err)
	}
	namesMap = updateNameList(namesMap, claims)
	logrus.Debugf("found %d new names from claims that were modified", len(namesMap)-prevNamesLength)
	prevNamesLength = len(namesMap)
	// CLAIMS THAT BECAME VALID SINCE [SELECT DISTINCT claim.name FROM claim WHERE claim.valid_at_height >= 852512;]
	claims, err = model.Claims(qm.Select("DISTINCT "+c.Name), model.ClaimWhere.ValidAtHeight.GTE(uint(lastSync.LastHeight))).AllG()
	if err != nil {
		return nil, errors.Err(err)
	}
	namesMap = updateNameList(namesMap, claims)
	logrus.Debugf("found %d new names from claims that became valid", len(namesMap)-prevNamesLength)
	var namesToFind []interface{}
	for name := range namesMap {
		namesToFind = append(namesToFind, name)
	}

	upTo = 5000
	var claimsToUpdate model.ClaimSlice
	for len(namesToFind) > 0 {
		if len(namesToFind) < upTo {
			upTo = len(namesToFind)
		}
		toFind := namesToFind[:upTo]
		claims, err := model.Claims(qm.Select(c.ID, c.Name), qm.WhereIn(c.Name+" IN ?", toFind...)).AllG()
		if err != nil {
			return nil, errors.Err(err)
		}
		claimsToUpdate = append(claimsToUpdate, claims...)
		logrus.Debugf("found %d additional claims from name list", len(claims))
		namesToFind = namesToFind[upTo:]
	}
	claimsToUpdate, err = populateClaimID(claimsToUpdate)
	return claimsToUpdate, err
}

func populateClaimID(originalClaims model.ClaimSlice) (model.ClaimSlice, error) {
	var idsToFind []interface{}
	var idsOfClaims []uint64
	IDMap := make(map[uint64]int)
	for i, claim := range originalClaims {
		idsToFind = append(idsToFind, claim.ID)
		idsOfClaims = append(idsOfClaims, claim.ID)
		IDMap[claim.ID] = i
	}
	upTo := 5000
	c := model.ClaimColumns
	for len(idsToFind) > 0 {
		if len(idsToFind) < upTo {
			upTo = len(idsToFind)
		}
		toFind := idsToFind[:upTo]
		claims, err := model.Claims(qm.Select(c.ID, c.ClaimID), qm.WhereIn(c.ID+" IN ?", toFind...)).AllG()
		if err != nil {
			return nil, errors.Err(err)
		}
		logrus.Debugf("found %d additional claims from claim_id list", len(claims))
		for _, claim := range claims {
			c := originalClaims[IDMap[claim.ID]]
			c.ClaimID = claim.ClaimID
			originalClaims[IDMap[claim.ID]] = c
		}
		idsToFind = idsToFind[upTo:]
	}
	return originalClaims, nil
}

func updateNameList(m map[string]bool, claims model.ClaimSlice) map[string]bool {
	if m == nil {
		m = make(map[string]bool)
	}
	for _, claim := range claims {
		if _, ok := m[claim.Name]; !ok {
			m[claim.Name] = true
		}
	}
	return m
}

func getSpentClaimsToUpdate(hasUpdate bool, lastProcessed uint64) (model.ClaimSlice, uint64, error) {
	w := model.OutputWhere
	o := model.OutputColumns
	outputMods := []qm.QueryMod{
		qm.Select(o.ID, o.IsSpent, o.TransactionHash),
		w.ModifiedAt.GTE(lastSync.PreviousSyncTime),
		w.IsSpent.EQ(true),
		w.ID.GT(lastProcessed),
		qm.Limit(50000),
	}
	var outputs model.OutputSlice
	var claims model.ClaimSlice
	var claimsToAdd model.ClaimSlice
	var err error
	outputs, err = model.Outputs(outputMods...).AllG()
	for len(outputs) > 0 {
		var txHashList []interface{}
		for _, o := range outputs {
			txHashList = append(txHashList, o.TransactionHash)
		}
		txHashCol := model.ClaimColumns.TransactionHashID
		if hasUpdate {
			txHashCol = model.ClaimColumns.TransactionHashUpdate
		}
		c := model.ClaimColumns
		claimsToAdd, err = model.Claims(qm.Select(c.ID, c.ClaimID, txHashCol), qm.WhereIn(txHashCol+" IN ?", txHashList...)).AllG()
		if err != nil {
			return nil, 0, errors.Err(err)
		}
		claims = append(claims, claimsToAdd...)
		logrus.Debug("outputs found: ", len(outputs), " claims found up to: ", len(claims))
		//nextOutputMods := append(outputMods, w.ID.GT(outputs[len(outputs)-1].ID))
		//outputs, err = model.Outputs(nextOutputMods...).AllG()
		//if err != nil {
		//	return nil, 0, errors.Err(err)
		//}
	}
	lastProcessed = outputs[len(outputs)-1].ID
	return claims, lastProcessed, nil
}

func updateSpentClaims() error {
	var lastProcessed uint64
	for {
		//Claims without updates
		claims, newLastProcessed, err := getSpentClaimsToUpdate(false, lastProcessed)
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
		if lastProcessed == newLastProcessed {
			break
		}
		lastProcessed = newLastProcessed
	}
	lastProcessed = 0
	for {
		//Claims without updates
		claims, newLastProcessed, err := getSpentClaimsToUpdate(true, lastProcessed)
		if err != nil {
			return err
		}
		for _, claim := range claims {
			claim.BidState = "Spent"
			claim.ModifiedAt = time.Now()
		}
		upTo := 1000
		logrus.Debugf("%d claims left to update", len(claims))
		for len(claims) > 0 {
			if len(claims) < upTo {
				upTo = len(claims)
			}
			toUpdate := claims[:upTo]
			args := []interface{}{time.Now()}
			for _, c := range toUpdate {
				args = append(args, c.ID)
			}
			updateQuery := fmt.Sprintf(`UPDATE claim SET bid_state="Spent", modified_at = ? WHERE id IN (%s)`, query.Qs(len(toUpdate)))
			if _, err := boil.GetDB().Exec(updateQuery, args...); err != nil {
				return err
			}
			logrus.Debugf("%d claims left to update", len(claims))
			claims = claims[upTo:]
		}
		if lastProcessed == newLastProcessed {
			break
		}
		lastProcessed = newLastProcessed
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
