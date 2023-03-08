package jobs

import (
	"runtime"
	"sync"
	"time"

	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/lbryio/chainquery/model"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

// const outputFixSyncJob = "outputfixsync"

// OutputFixSync this will check for spent claims and update them if they are incorrectly marked spent.
func OutputFixSync() {
	err := fixOutputs()
	if err != nil {
		logrus.Error("Fix Outputs Sync:", errors.FullTrace(err))
	}
}

func fixOutputs() error {
	wg := sync.WaitGroup{}
	spentClaimsChan := make(chan *model.Claim, 100)
	errorsChan := make(chan error, runtime.NumCPU())
	c := model.ClaimColumns
	for i := 0; i < runtime.NumCPU()-1; i++ {
		wg.Add(1)
		go func(spentClaims chan *model.Claim, errorsChan chan error) {
			defer wg.Done()
			for claim := range spentClaims {
				where := model.OutputWhere
				isSpent, err := model.Outputs(
					where.ClaimID.EQ(null.StringFrom(claim.ClaimID)),
					where.TransactionHash.EQ(claim.TransactionHashUpdate.String),
					where.Vout.EQ(claim.VoutUpdate.Uint),
					where.IsSpent.EQ(true)).ExistsG()
				if err != nil {
					errorsChan <- errors.Err(err)
					return
				}
				if !isSpent {
					logrus.Debugf("%s is not really spent", claim.ClaimID)
					claim.BidState = "Active"
					claim.ModifiedAt = time.Now()
					err := claim.UpdateG(boil.Whitelist(c.BidState, c.ModifiedAt))
					if err != nil {
						errorsChan <- errors.Err(err)
						return
					}
				}
			}
		}(spentClaimsChan, errorsChan)
	}

	lastClaimRecordID := uint64(0)
	claims, err := model.Claims(
		qm.Select(c.ID, c.ClaimID, c.TransactionHashID, c.TransactionHashUpdate, c.VoutUpdate),
		model.ClaimWhere.BidState.EQ("Spent"),
		model.ClaimWhere.ID.GT(lastClaimRecordID),
		qm.Limit(15000)).AllG()
	if err != nil {
		return errors.Err(err)
	}

	for len(claims) != 0 {
		logrus.Debugf("enqueing claims from %d to %d", claims[0].ID, claims[len(claims)-1].ID)
		for _, claim := range claims {
			spentClaimsChan <- claim
			lastClaimRecordID = claim.ID
		}
		claims, err = model.Claims(
			qm.Select(c.ID, c.ClaimID, c.TransactionHashID, c.TransactionHashUpdate, c.VoutUpdate),
			model.ClaimWhere.BidState.EQ("Spent"),
			model.ClaimWhere.ID.GT(lastClaimRecordID),
			qm.Limit(15000)).AllG()
		if err != nil {
			return errors.Err(err)
		}
	}
	close(spentClaimsChan)
	wg.Wait()
	close(errorsChan)
	for e := range errorsChan {
		logrus.Errorf("a worker incurrred in an error: %s", e.Error())
	}
	return nil
}
