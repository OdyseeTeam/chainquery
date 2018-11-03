package main

//go:generate go-bindata -nometadata -o migration/bindata.go -pkg migration -ignore bindata.go migration/
//go:generate go fmt ./migration/bindata.go
//go:generate goimports -l ./migration/bindata.go

import (
	"net/http"
	"os"

	"github.com/lbryio/chainquery/apiactions"
	"github.com/lbryio/chainquery/config"
	"github.com/lbryio/chainquery/daemon"
	"github.com/lbryio/chainquery/db"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/meta"
	"github.com/lbryio/chainquery/swagger/apiserver"
	"github.com/lbryio/chainquery/twilio"

	log "github.com/sirupsen/logrus"
)

func main() {
	config.InitializeConfiguration()
	config.InitSlack()
	twilio.InitTwilio()
	apiactions.AutoUpdateCommand = config.GetAutoUpdateCommand()
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	http.DefaultClient.Timeout = config.GetDefaultClientTimeout()

	if len(os.Args) < 2 {
		return
	}

	command := os.Args[1]
	switch command {
	default:
		log.Errorf("Invalid command: '%s'\n", command)
	case "version":
		println("Version: " + meta.GetVersion())
		println("Version(long): " + meta.GetVersionLong())
	case "serve":
		//Main Chainquery DB connection
		dbInstance, err := db.Init(config.GetMySQLDSN(), config.GetDebugQueryMode())
		if err != nil {
			log.Panic(err)
		}
		defer db.CloseDB(dbInstance)

		lbrycrdClient := lbrycrd.Init()
		defer lbrycrdClient.Shutdown()

		go swagger.InitApiServer(config.GetAPIHostAndPort())
		daemon.DoYourThing()
	}
}
