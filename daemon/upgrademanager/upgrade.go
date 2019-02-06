package upgrademanager

import (
	"time"

	"github.com/lbryio/chainquery/daemon/jobs"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/errors"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

const (
	appVersion  = 12
	apiVersion  = 12
	dataVersion = 12
)

// RunUpgradesForVersion - Migrations are for structure of the data. Upgrade Manager scripts are for the data itself.
// Since it could take hours to rebuild the data chainquery is designed to reprocess data so
// that there is no downtime of data availability since many system will be using it.
func RunUpgradesForVersion() {
	var appStatus *model.ApplicationStatus
	var err error
	if !model.ApplicationStatusExistsGP(1) {
		appStatus = &model.ApplicationStatus{AppVersion: appVersion, APIVersion: apiVersion, DataVersion: dataVersion}
		if err := appStatus.InsertG(boil.Infer()); err != nil {
			err := errors.Prefix("App Status Error: ", err)
			panic(err)
		}
	} else {
		appStatus, err = model.FindApplicationStatusG(1)
		if err != nil {
			logrus.Error("Application cannot be upgraded: ", err)
			return
		}
		////Run all upgrades and let it determine if they should execute
		//
		////upgrade123(appStatus.appVersion)
		upgradeFrom1(appStatus.AppVersion)
		upgradeFrom2(appStatus.AppVersion)
		upgradeFrom3(appStatus.AppVersion)
		upgradeFrom4(appStatus.AppVersion)
		upgradeFrom5(appStatus.AppVersion)
		upgradeFrom6(appStatus.AppVersion)
		upgradeFrom7(appStatus.AppVersion)
		upgradeFrom8(appStatus.AppVersion)
		upgradeFrom9(appStatus.AppVersion)
		upgradeFrom10(appStatus.AppVersion)
		////Increment and save
		//
		logrus.Debug("Upgrading app status version to App-", appVersion, " Data-", dataVersion, " Api-", apiVersion)
		appStatus.AppVersion = appVersion
		appStatus.DataVersion = dataVersion
		appStatus.APIVersion = apiVersion
	}
	if err := appStatus.UpdateG(boil.Infer()); err != nil {
		err := errors.Prefix("App Status Error: ", err)
		panic(err)
	}
	logrus.Debug("All necessary upgrades are finished!")
}

//
//func upgradeFrom123(version int){
//	util.TimeTrack(time.Now(),"script DoThis","always")
//	if version < 123{
//		scriptDoThis()
//	}
//}

func upgradeFrom1(version int) {
	if version < 2 {
		logrus.Info("Re-Processing all claim outputs")
		reProcessAllClaims()
	}
}

func upgradeFrom2(version int) {
	if version < 3 {
		logrus.Info("Re-Processing all claim outputs")
		reProcessAllClaims()
	}
}

func upgradeFrom3(version int) {
	if version < 4 {
		logrus.Info("Re-Processing all claim outputs")
		reProcessAllClaims()
	}
}

func upgradeFrom4(version int) {
	if version < 5 {
		logrus.Info("Updating Claims with ClaimAddress")
		setClaimAddresses()
	}
}

func upgradeFrom5(version int) {
	if version < 6 {
		logrus.Info("Deleting top 50 blocks to ensure consistency for release")
		highestBlock, err := model.Blocks(qm.OrderBy(model.BlockColumns.Height + " DESC")).OneG()
		if err != nil {
			logrus.Error(err)
			return
		}
		blocks, err := model.Blocks(qm.Where(model.BlockColumns.Height+">= ?", highestBlock.Height-50)).AllG()
		if err != nil {
			logrus.Error(err)
			return
		}
		if err := blocks.DeleteAllG(); err != nil {
			logrus.Error(err)
			return
		}
	}
}

func upgradeFrom6(version int) {
	if version < 7 {
		logrus.Info("Setting the height of all claims")
		setBlockHeightOnAllClaims()
	}
}

func upgradeFrom7(version int) {
	if version < 8 {
		block, err := model.Blocks(qm.OrderBy(model.BlockColumns.Height+" DESC"), qm.Limit(1)).OneG()
		if err != nil {
			logrus.Error(err)
			return
		}
		if block != nil && block.Height < 400155 {
			logrus.Info("Reprocessing all claims equal to or above height 400155")
			reProcessAllClaimsFromHeight(400155) // https://github.com/lbryio/lbrycrd/pull/137 - expiration hardfork.
		}

	}
}

func upgradeFrom8(version int) {
	if version < 9 {
		logrus.Info("Re-Processing all claim outputs")
		reProcessAllClaims()
	}
}

func upgradeFrom9(version int) {
	if version < 10 {
		//Get blockheight for calculating expired status
		count, err := lbrycrd.GetBlockCount()
		if err != nil {
			logrus.Error("Upgrade 9-10: Error getting block height", err)
			return
		}
		expiredClaims, err := model.Claims(qm.Where(model.ClaimColumns.BidState+"=?", "Expired")).AllG()
		if err != nil {
			logrus.Error("Upgrade 9-10: ", err)
		}

		for _, expiredClaim := range expiredClaims {
			if !jobs.GetIsExpiredAtHeight(expiredClaim.Height, uint(*count)) {
				expiredClaim.BidState = "Active" //Will update it to go through the claimtrie sync
				expiredClaim.ModifiedAt = time.Now()
				err = expiredClaim.UpdateG(boil.Infer())
				if err != nil {
					logrus.Error("Upgrade 9-10: ", err)
				}
			}
		}
	}
}

func upgradeFrom10(version int) {
	if version < 10 {
		logrus.Info("Re-Processing all claim outputs")
		reProcessAllClaims()
	}
}

func upgradeFrom11(version int) {
	if version < 12 {
		logrus.Info("Re-Processing all claim outputs")
		reProcessAllClaims()
	}
}
