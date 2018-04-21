package main

//go:generate go-bindata -o migration/bindata.go -pkg migration -ignore bindata.go migration/

import (
	"net/http"
	"os"
	"strconv"

	"github.com/lbryio/chainquery/config"
	"github.com/lbryio/chainquery/daemon"
	"github.com/lbryio/chainquery/db"
	"github.com/lbryio/chainquery/lbrycrd"

	log "github.com/sirupsen/logrus"
)

var DebugMode bool

func main() {
	config.InitializeConfiguration()
	//defer profile.Start(profile.ProfilePath(os.Getenv("HOME") + "/chainquery")).Stop()

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	http.DefaultClient.Timeout = config.GetDefaultClientTimeout()

	if len(os.Args) < 2 {
		return
	}

	DebugMode, err := strconv.ParseBool(os.Getenv("DEBUGGING"))
	if err != nil {
		DebugMode = false
	}
	if DebugMode {
		log.SetLevel(log.DebugLevel)
	}

	command := os.Args[1]
	switch command {
	default:
		log.Errorf("Invalid command: '%s'\n", command)
	case "version":
		println("ALPHA")
	case "serve":
		dbInstance, err := db.Init(config.GetMySQLDSN(), DebugMode)
		if err != nil {
			panic(err)
		}
		defer dbInstance.Close()

		lbrycrdClient := lbrycrd.Init()
		defer lbrycrdClient.Shutdown()

		_, err = lbrycrd.GetBalance()
		if err != nil {
			log.Panicf("Error connecting to lbrycrd: %+v", err)
		}
		daemon.DoYourThing()
	}
}
