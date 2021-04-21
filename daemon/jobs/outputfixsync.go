package jobs

import (
	"time"

	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/lbryio/chainquery/model"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
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
	lastClaimRecordID := uint64(0)
	c := model.ClaimColumns
	claims, err := model.Claims(
		qm.Select(c.ID, c.ClaimID, c.TransactionHashID, c.TransactionHashUpdate, c.VoutUpdate),
		model.ClaimWhere.BidState.EQ("Spent"),
		model.ClaimWhere.ID.GT(lastClaimRecordID),
		qm.Limit(1000)).AllG()
	if err != nil {
		return errors.Err(err)
	}
	if len(claims) == 0 {
		return nil
	}
	logrus.Debugf("check claim from %d to %d", claims[0].ID, claims[len(claims)-1].ID)
	for len(claims) != 0 {
		for _, claim := range claims {
			lastClaimRecordID = claim.ID
			where := model.OutputWhere
			isSpent, err := model.Outputs(
				where.ClaimID.EQ(null.StringFrom(claim.ClaimID)),
				where.TransactionHash.EQ(claim.TransactionHashUpdate.String),
				where.Vout.EQ(claim.VoutUpdate.Uint),
				where.IsSpent.EQ(true)).ExistsG()
			if err != nil {
				return errors.Err(err)
			}
			if !isSpent {
				logrus.Debugf("%s is not really spent", claim.ClaimID)
				claim.BidState = "Active"
				claim.ModifiedAt = time.Now()
				err := claim.UpdateG(boil.Whitelist(c.BidState, c.ModifiedAt))
				if err != nil {
					return errors.Err(err)
				}
			}
		}
		claims, err = model.Claims(
			qm.Select(c.ID, c.ClaimID, c.TransactionHashID, c.TransactionHashUpdate, c.VoutUpdate),
			model.ClaimWhere.BidState.EQ("Spent"),
			model.ClaimWhere.ID.GT(lastClaimRecordID),
			qm.Limit(1000)).AllG()
		if err != nil {
			return errors.Err(err)
		}
	}
	return nil
}
