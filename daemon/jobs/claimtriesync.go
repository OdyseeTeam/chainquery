package jobs

import (
	"github.com/lbryio/chainquery/lbrycrd"

	"github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/model"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"runtime"
)

func ClaimTrieSync() {
	logrus.Info("ClaimTrie sync started... ")
	client := lbrycrd.DefaultClient()
	names, err := client.GetClaimsInTrie()
	if err != nil {
		panic(err) //
	}
	processingQueue := make(chan lbrycrd.Claim)
	initWorkers(runtime.NumCPU()-1, processingQueue)
	println("size: ", len(names))
	for _, claimedName := range names {
		claims, err := client.GetClaimsForName(claimedName.Name)
		if err != nil {
			panic(err)
		}
		for _, claimJSON := range claims.Claims {
			processingQueue <- claimJSON
		}
	}
}

func getClaimStatus(claim *model.Claim) string {
	return "Accepted"
}

func initWorkers(nrWorkers int, jobs <-chan lbrycrd.Claim) {
	for i := 0; i < nrWorkers; i++ {
		go processor(jobs)
	}
}

func processor(jobs <-chan lbrycrd.Claim) error {
	for job := range jobs {
		syncClaim(&job)
	}
	return nil
}

func syncClaim(claimJSON *lbrycrd.Claim) {
	claim := datastore.GetClaim(claimJSON.ClaimId)
	if claim == nil {
		unknown, _ := model.UnknownClaimsG(qm.Where(model.UnknownClaimColumns.ClaimID+"=?", claimJSON.ClaimId)).One()
		if unknown == nil {
			logrus.Error("Missing Claim: ", claimJSON.ClaimId, " ", claimJSON.TxId, " ", claimJSON.N)
		}
		return
	}
	claim.ValidAtHeight = uint(claimJSON.ValidAtHeight)
	claim.EffectiveAmount = claimJSON.EffectiveAmount
	claim.BidState = getClaimStatus(claim)
	datastore.PutClaim(claim)
}
