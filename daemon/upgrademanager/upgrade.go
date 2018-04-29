package upgrademanager

import (
	//"github.com/sirupsen/logrus"
	//"github.com/lbryio/chainquery/util"
	//"time"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/errors"
	"github.com/sirupsen/logrus"
)

const (
	AppVersion  = 2
	ApiVersion  = 2
	DataVersion = 2
)

//Migrations are for structure of the data. Upgrade Manager scripts are for the data itself.
// Since it could take hours to rebuild the data chainquery is designed to reprocess data so
// that there is no downtime of data availability since many system will be using it.
func RunUpgradesForVersion() {
	var appStatus *model.ApplicationStatus
	var err error
	if !model.ApplicationStatusExistsGP(1) {
		appStatus = &model.ApplicationStatus{AppVersion: 1, APIVersion: 1, DataVersion: 1}
		if err := appStatus.InsertG(); err != nil {
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
		////upgrade_123(appStatus.AppVersion)
		upgradeFrom_1(appStatus.AppVersion)
		////Increment and save
		//
		logrus.Info("Upgrading app status version to App-", AppVersion, " Data-", DataVersion, " Api-", ApiVersion)
		appStatus.AppVersion = AppVersion
		appStatus.DataVersion = DataVersion
		appStatus.APIVersion = ApiVersion
	}
	if err := appStatus.UpdateG(); err != nil {
		err := errors.Prefix("App Status Error: ", err)
		panic(err)
	}
	logrus.Info("All necessary upgrades are finished!")
}

//
//func upgradeFrom_123(version int){
//	util.TimeTrack(time.Now(),"script DoThis","always")
//	if version < 123{
//		scriptDoThis()
//	}
//}

func upgradeFrom_1(version int) {
	if version < 2 {
		logrus.Info("Re-Processing all claim outputs")
		reProcessAllClaims()
	}
}
