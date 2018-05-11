package main

//go:generate go-bindata -nometadata -o migration/bindata.go -pkg migration -ignore bindata.go migration/
//go:generate go fmt ./migration/bindata.go
//go:generate goimports -l ./migration/bindata.go

import (
	"net/http"
	"os"

	"github.com/lbryio/chainquery/config"
	"github.com/lbryio/chainquery/daemon"
	"github.com/lbryio/chainquery/db"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/swagger/apiserver"

	"github.com/lbryio/chainquery/apiactions"
	log "github.com/sirupsen/logrus"
)

func main() {
	config.InitializeConfiguration()
	config.InitSlack()
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
		println("v0.1.6")
	case "serve":
		//Main Chainquery DB connection
		dbInstance, err := db.Init(config.GetMySQLDSN(), config.GetDebugMode())
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
