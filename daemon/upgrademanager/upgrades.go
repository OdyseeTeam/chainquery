package upgrademanager

import (
	//"github.com/sirupsen/logrus"
	//"github.com/lbryio/chainquery/util"
	//"time"
	"github.com/lbryio/chainquery/model"
	"github.com/sirupsen/logrus"
)

const (
	AppVersion  = 1
	ApiVersion  = 1
	DataVersion = 1
)

//Migrations are for structure of the data. Upgrade Manager scripts are for the data itself.
// Since it could take hours to rebuild the data chainquery is designed to reprocess data so
// that there is no downtime of data availability since many system will be using it.
func RunUpgradesForVersion() {
	var appStatus *model.ApplicationStatus
	var err error
	if !model.ApplicationStatusExistsGP(1) {
		appStatus = &model.ApplicationStatus{}
	} else {
		appStatus, err = model.ApplicationStatusesG().One()
		if err != nil {
			logrus.Error("Application cannot be upgraded: ", err)
			return
		}
		if appStatus != nil {
			appStatus = &model.ApplicationStatus{}
		}
		////Run all upgrades and let it determine if they should execute
		//
		////upgrade_123(appStatus.AppVersion)
		//
		////Increment and save
		//
		//if appStatus.ID == 0 {
		//	appStatus.AppVersion = AppVersion
		//	appStatus.InsertG()
		//} else {
		//	appStatus.AppVersion = AppVersion
		//	appStatus.UpdateG()
		//}
	}
	logrus.Info("All necessary upgrades are finished!")
}

//
//func upgradeFrom_123(int version){
//	util.TimeTrack(time.Now(),"script DoThis","always")
//	if version < 123{
//		scriptDoThis()
//	}
//}
