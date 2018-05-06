package jobs

import (
	"runtime"
	"strconv"
	"sync"

	"github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/chainquery/util"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"time"
)

const claimTrieSyncJob = "claimtriesyncjob"

var blockHeight uint64
var blocksToExpiration uint = 262974 //Hardcoded! https://lbry.io/faq/claimtrie-implementation

// ClaimTrieSync synchronizes claimtrie information that is calculated and enforced by lbrycrd.
func ClaimTrieSync() {
	defer util.TimeTrack(time.Now(), "ClaimTrieSync", "always")
	logrus.Info("ClaimTrieSync: started... ")
	jobStatus, err := getClaimTrieSyncJobStatus()
	if err != nil {
		logrus.Error(err)
	}
	updatedClaims, err := getUpdatedClaims(jobStatus)
	if err != nil {
		saveJobError(jobStatus, err)
		panic(err)
	}
	if len(updatedClaims) == 0 {
		logrus.Info("ClaimTrieSync: All claims are up to date :)")
		return
	}
	logrus.Info("Claims to update " + strconv.Itoa(len(updatedClaims)))

	//Get blockheight for calculating expired status
	count, err := lbrycrd.GetBlockCount()
	if err != nil {
		panic(err)
	}
	blockHeight = *count

	//For syncing the claims
	if err := syncClaims(updatedClaims); err != nil {
		saveJobError(jobStatus, err)
	}

	//For Setting Controlling Claims
	if err := setControllingClaimForNames(updatedClaims); err != nil {
		saveJobError(jobStatus, err)
	}
	jobStatus.LastSync = time.Now()
	if err := jobStatus.UpdateG(); err != nil {
		panic(err)
	}
	logrus.Info("ClaimTrieSync: Processed " + strconv.Itoa(len(updatedClaims)) + " claims.")
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
	logrus.Info("ClaimTrieSync: controlling claim status update started... ")
	controlwg := sync.WaitGroup{}
	setControllingQueue := make(chan string, 100)
	initControllingWorkers(runtime.NumCPU()-1, setControllingQueue, &controlwg)
	for _, claim := range claims {
		setControllingQueue <- claim.Name
	}
	close(setControllingQueue)
	controlwg.Wait()
	logrus.Info("ClaimTrieSync: controlling claim status update complete... ")

	return nil
}

func setControllingClaimForName(name string) {
	claim, _ := model.ClaimsG(
		qm.Where(model.ClaimColumns.Name+"=?", name),
		qm.And(model.ClaimColumns.BidState+"=?", "Active"),
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

	logrus.Info("ClaimTrieSync: claim  update started... ")
	syncwg := sync.WaitGroup{}
	processingQueue := make(chan lbrycrd.Claim, 100)
	initSyncWorkers(runtime.NumCPU()-1, processingQueue, &syncwg)
	for i, claim := range claims {
		if i%1000 == 0 {
			logrus.Info("syncing ", i, " of ", len(claims), " queued - ", len(processingQueue))
		}
		claims, err := lbrycrd.GetClaimsForName(claim.Name)
		if err != nil {
			logrus.Error("Could not get claims for name: ", claim.Name, " Error: ", err)
		}
		for _, claimJSON := range claims.Claims {
			processingQueue <- claimJSON
		}
	}
	close(processingQueue)
	syncwg.Wait()

	logrus.Info("ClaimTrieSync: claim  update complete... ")

	return nil
}

func syncClaim(claimJSON *lbrycrd.Claim) {
	hasChanges := false
	claim := datastore.GetClaim(claimJSON.ClaimID)
	if claim == nil {
		unknown, _ := model.UnknownClaimsG(qm.Where(model.UnknownClaimColumns.ClaimID+"=?", claimJSON.ClaimID)).One()
		if unknown == nil {
			logrus.Debug("Missing Claim: ", claimJSON.ClaimID, " ", claimJSON.TxID, " ", claimJSON.N)
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
	transaction := claim.TransactionByHashG().OneP()
	output := transaction.OutputsG(qm.Where(model.OutputColumns.Vout+"=?", claim.Vout)).OneP()
	spend, _ := output.SpentByInputG().One()
	if spend != nil {
		status = "Spent"
	}
	height := claim.Height
	if height+blocksToExpiration > uint(blockHeight) {
		status = "Expired"
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
	clause := qm.SQL(`
		SELECT 
					` + claimIDCol + `, 
					` + claimNameCol + `
		FROM 		` + model.TableNames.Claim + ` 
		LEFT JOIN 	` + model.TableNames.Support + `  
			ON 		` + supportedIDCol + "=" + claimIDCol + `  
		WHERE 		` + supportModifiedCol + ">=" + lastSyncStr + ` 
		OR 			` + claimModifiedCol + ">=" + lastSyncStr + `  
		GROUP BY 	` + claimIDCol + `,` + claimNameCol)

	return model.ClaimsG(clause).All()

}

func getClaimTrieSyncJobStatus() (*model.JobStatus, error) {
	jobStatus, _ := model.FindJobStatusG(claimTrieSyncJob)
	if jobStatus == nil {
		jobStatus = &model.JobStatus{JobName: claimTrieSyncJob, LastSync: time.Time{}}
		if err := jobStatus.InsertG(); err != nil {
			return nil, err
		}
	}

	return jobStatus, nil
}

func saveJobError(jobStatus *model.JobStatus, err error) {
	jobStatus.ErrorMessage.String = err.Error()
	jobStatus.ErrorMessage.Valid = true
	cols := model.JobStatusColumns
	if err := jobStatus.UpsertG([]string{cols.JobName, cols.LastSync, cols.IsSuccess, cols.ErrorMessage}); err != nil {
		logrus.Error(err)
	}
}
