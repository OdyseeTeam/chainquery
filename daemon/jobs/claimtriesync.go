package jobs

import (
	"runtime"
	"strconv"
	"sync"

	"github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"

	"github.com/lbryio/lbry.go/errors"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"time"
)

const claimTrieSyncJob = "claimtriesyncjob"

var expirationHardForkHeight uint = 400155    // https://github.com/lbryio/lbrycrd/pull/137
var hardForkBlocksToExpiration uint = 2102400 // https://github.com/lbryio/lbrycrd/pull/137
var blockHeight uint64
var blocksToExpiration uint = 262974 //Hardcoded! https://lbry.io/faq/claimtrie-implementation
// ClaimTrieSyncRunning is a variable used to show whether or not the job is running already.
var claimTrieSyncRunning = false

// ClaimTrieSync synchronizes claimtrie information that is calculated and enforced by lbrycrd.
func ClaimTrieSync() {
	if !claimTrieSyncRunning {
		claimTrieSyncRunning = true
		//defer util.TimeTrack(time.Now(), "ClaimTrieSync", "always")
		logrus.Debug("ClaimTrieSync: started... ")
		jobStatus, err := getClaimTrieSyncJobStatus()
		if err != nil {
			logrus.Error(err)
		}

		//For Updating claims that are spent ( no longer in claimtrie )
		if err := updateSpentClaims(); err != nil {
			logrus.Error("ClaimTrieSync:", err)
			saveJobError(jobStatus, err)
		}

		started := time.Now()

		updatedClaims, err := getUpdatedClaims(jobStatus)
		if err != nil {
			logrus.Error("ClaimTrieSync:", err)
			saveJobError(jobStatus, err)
			panic(err)
		}
		if len(updatedClaims) == 0 {
			logrus.Debug("ClaimTrieSync: All claims are up to date :)")
			return
		}
		logrus.Debug("ClaimTrieSync: Claims to update " + strconv.Itoa(len(updatedClaims)))

		//Get blockheight for calculating expired status
		count, err := lbrycrd.GetBlockCount()
		if err != nil {
			panic(err)
		}
		blockHeight = *count

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
		if err := jobStatus.UpdateG(); err != nil {
			panic(err)
		}
		logrus.Debug("ClaimTrieSync: Processed " + strconv.Itoa(len(updatedClaims)) + " claims.")
		claimTrieSyncRunning = false
	}
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
		setControllingClaimForName(name)
	}
}

func setControllingClaimForNames(claims model.ClaimSlice) error {
	logrus.Debug("ClaimTrieSync: controlling claim status update started... ")
	controlwg := sync.WaitGroup{}
	setControllingQueue := make(chan string, 100)
	initControllingWorkers(runtime.NumCPU()-1, setControllingQueue, &controlwg)
	for _, claim := range claims {
		setControllingQueue <- claim.Name
	}
	close(setControllingQueue)
	controlwg.Wait()
	logrus.Debug("ClaimTrieSync: controlling claim status update complete... ")

	return nil
}

func setControllingClaimForName(name string) {
	claim, _ := model.ClaimsG(
		qm.Where(model.ClaimColumns.Name+"=?", name),
		qm.Where(model.ClaimColumns.BidState+"!=?", "Spent"),
		qm.OrderBy(model.ClaimColumns.ValidAtHeight+" DESC")).One()

	if claim != nil {
		if claim.BidState != "Controlling" {

			claim.BidState = "Controlling"

			err := datastore.PutClaim(claim)
			if err != nil {
				panic(err)
			}
		}
	}
}

func syncClaims(claims model.ClaimSlice) error {

	logrus.Debug("ClaimTrieSync: claim  update started... ")
	syncwg := sync.WaitGroup{}
	processingQueue := make(chan lbrycrd.Claim, 100)
	initSyncWorkers(runtime.NumCPU()-1, processingQueue, &syncwg)
	for i, claim := range claims {
		if i%1000 == 0 {
			logrus.Debug("ClaimTrieSync: syncing ", i, " of ", len(claims), " queued - ", len(processingQueue))
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

	logrus.Debug("ClaimTrieSync: claim  update complete... ")

	return nil
}

func syncClaim(claimJSON *lbrycrd.Claim) {
	hasChanges := false
	claim := datastore.GetClaim(claimJSON.ClaimID)
	if claim == nil {
		unknown, _ := model.UnknownClaimsG(qm.Where(model.UnknownClaimColumns.ClaimID+"=?", claimJSON.ClaimID)).One()
		if unknown == nil {
			logrus.Debug("ClaimTrieSync: Missing Claim ", claimJSON.ClaimID, " ", claimJSON.TxID, " ", claimJSON.N)
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
	status := getClaimStatus(claim)
	if claim.BidState != status {
		claim.BidState = getClaimStatus(claim)
		hasChanges = true
	}
	if hasChanges {
		if err := datastore.PutClaim(claim); err != nil {
			logrus.Error("ClaimTrieSync: unable to sync claim ", claim.ClaimID, ". JSON-", claimJSON)
			logrus.Debug("Error: ", err)
		}
	}
}

func getClaimStatus(claim *model.Claim) string {
	status := "Accepted"
	//Transaction and output should never be missing if the claim exists.
	transaction, err := claim.TransactionByHashG().One()
	if err != nil {
		logrus.Error("could not find transaction ", claim.TransactionByHashID, " : ", err)
		return status
	}

	output, err := transaction.OutputsG(qm.Where(model.OutputColumns.Vout+"=?", claim.Vout)).One()
	if err != nil {
		logrus.Error("could not find output ", claim.TransactionByHashID, "-", claim.Vout, " : ", err)
		return status
	}

	if output.IsSpent {
		status = "Spent" //Should be unreachable because claim would be out of claimtrie if spent.
	}
	height := claim.Height
	if height >= expirationHardForkHeight {
		// https://github.com/lbryio/lbrycrd/pull/137 - HardFork extends claim expiration.
		if height+hardForkBlocksToExpiration < uint(blockHeight) {
			status = "Expired"
		}
	} else {
		if height+blocksToExpiration < uint(blockHeight) {
			status = "Expired"
		}
	}

	//Neither Spent or Expired = Active
	if status == "Accepted" {
		status = "Active"
	}

	return status
}

func getUpdatedClaims(jobStatus *model.JobStatus) (model.ClaimSlice, error) {
	claimIDCol := model.TableNames.Claim + "." + model.ClaimColumns.ClaimID
	claimNameCol := model.TableNames.Claim + "." + model.ClaimColumns.Name
	supportedIDCol := model.TableNames.Support + "." + model.SupportColumns.SupportedClaimID
	supportModifiedCol := model.TableNames.Support + ".modified" //+model.SupportColumns.Modified
	claimModifiedCol := model.TableNames.Claim + "." + model.ClaimColumns.Modified
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
			GROUP BY  claim.name
		)
`
	return model.ClaimsG(qm.SQL(query)).All()

}

func getSpentClaimsToUpdate() (model.ClaimSlice, error) {

	claim := model.TableNames.Claim

	claimID := claim + "." + model.ClaimColumns.ID
	claimClaimID := claim + "." + model.ClaimColumns.ClaimID
	claimTxByHash := claim + "." + model.ClaimColumns.TransactionByHashID
	claimVout := claim + "." + model.ClaimColumns.Vout
	claimBidState := claim + "." + model.ClaimColumns.BidState

	output := model.TableNames.Output
	outputTxHash := output + "." + model.OutputColumns.TransactionHash
	outputVout := output + "." + model.OutputColumns.Vout
	outputIsSpent := output + "." + model.OutputColumns.IsSpent

	clause := qm.SQL(`
		SELECT `+claimClaimID+`,`+claimID+` 
		FROM `+claim+` 
		INNER JOIN `+output+` 
			ON `+outputTxHash+` = `+claimTxByHash+` 
				AND `+outputVout+` = `+claimVout+` 
				AND `+outputIsSpent+` = ? 
		WHERE `+claimBidState+` != ? `, 1, "Spent")

	return model.ClaimsG(clause).All()
}

func updateSpentClaims() error {
	claims, err := getSpentClaimsToUpdate()
	if err != nil {
		return err
	}
	for _, claim := range claims {
		claim.BidState = "Spent"
		claim.Modified = time.Now()
		if err := claim.UpdateG(model.ClaimColumns.BidState, model.ClaimColumns.Modified); err != nil {
			return err
		}
	}
	return nil
}

func getClaimTrieSyncJobStatus() (*model.JobStatus, error) {
	jobStatus, _ := model.FindJobStatusG(claimTrieSyncJob)
	if jobStatus == nil {
		jobStatus = &model.JobStatus{JobName: claimTrieSyncJob, LastSync: time.Time{}}
		if err := jobStatus.InsertG(); err != nil {
			logrus.Panic("Cannot Retrieve/Create JobStatus for " + claimTrieSyncJob)
		}
	}

	return jobStatus, nil
}

func saveJobError(jobStatus *model.JobStatus, error error) {
	jobStatus.ErrorMessage.String = error.Error()
	jobStatus.ErrorMessage.Valid = true
	jobStatus.IsSuccess = false
	cols := model.JobStatusColumns
	if err := jobStatus.UpsertG([]string{cols.JobName, cols.IsSuccess, cols.ErrorMessage}); err != nil {
		logrus.Error(errors.Prefix("Saving Job Error Message "+error.Error(), err))
	}
}
