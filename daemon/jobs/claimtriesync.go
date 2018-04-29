package jobs

import (
	"runtime"
	"strconv"
	"sync"

	"github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

var blockHeight uint64
var blocksToExpiration uint = 262974 //Hardcoded! https://lbry.io/faq/claimtrie-implementation

func ClaimTrieSync() {
	logrus.Info("ClaimTrieSync: started... ")
	count, err := lbrycrd.GetBlockCount()
	if err != nil {
		panic(err)
	}
	blockHeight = *count
	names, err := lbrycrd.GetClaimsInTrie()
	if err != nil {
		panic(err)
	}
	//For syncing the claims
	logrus.Info("ClaimTrieSync: claim  update started... ")
	syncwg := sync.WaitGroup{}
	processingQueue := make(chan lbrycrd.Claim, 100)
	initSyncWorkers(runtime.NumCPU()-1, processingQueue, &syncwg)
	for _, claimedName := range names {
		claims, err := lbrycrd.GetClaimsForName(claimedName.Name)
		if err != nil {
			logrus.Error("Could not get claims for name: ", claimedName.Name, " Error: ", err)
		}
		for _, claimJSON := range claims.Claims {
			processingQueue <- claimJSON
		}
	}
	syncwg.Wait()
	close(processingQueue)
	logrus.Info("ClaimTrieSync: claim  update complete... ")

	//For Setting Controlling Claims
	logrus.Info("ClaimTrieSync: controlling claim status update started... ")
	controlwg := sync.WaitGroup{}
	setControllingQueue := make(chan string, 100)
	initControllingWorkers(runtime.NumCPU()-1, setControllingQueue, &controlwg)
	for _, claimedName := range names {
		setControllingQueue <- claimedName.Name
	}
	controlwg.Wait()
	close(setControllingQueue)
	logrus.Info("ClaimTrieSync: controlling claim status update complete... ")
	logrus.Info("ClaimTrieSync: Processed " + strconv.Itoa(len(names)) + " claimed names.")
}

func initSyncWorkers(nrWorkers int, jobs <-chan lbrycrd.Claim, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := 0; i < nrWorkers; i++ {
		wg.Add(1)
		go syncProcessor(jobs)
	}
}

func initControllingWorkers(nrWorkers int, jobs <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := 0; i < nrWorkers; i++ {
		wg.Add(1)
		go controllingProcessor(jobs)
	}
}

func syncProcessor(jobs <-chan lbrycrd.Claim) {
	for job := range jobs {
		syncClaim(&job)
	}
}

func controllingProcessor(names <-chan string) {
	for name := range names {
		setControllingClaimForName(name)
	}
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

func syncClaim(claimJSON *lbrycrd.Claim) {
	hasChanges := false
	claim := datastore.GetClaim(claimJSON.ClaimId)
	if claim == nil {
		unknown, _ := model.UnknownClaimsG(qm.Where(model.UnknownClaimColumns.ClaimID+"=?", claimJSON.ClaimId)).One()
		if unknown == nil {
			logrus.Debug("Missing Claim: ", claimJSON.ClaimId, " ", claimJSON.TxId, " ", claimJSON.N)
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
			logrus.Error("ClaimTrieSync: ", err)
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
