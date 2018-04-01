package main

//go:generate go-bindata -o migration/bindata.go -pkg migration -ignore bindata.go migration/

import (
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/lbryio/chainquery/daemon"
	"github.com/lbryio/chainquery/db"
	"github.com/lbryio/chainquery/lbrycrd"

	//"github.com/pkg/profile"
	"github.com/lbryio/chainquery/config"
	log "github.com/sirupsen/logrus"
)

var DebugMode bool

func main() {
	config.InitializeConfiguration()
	//defer profile.Start(profile.ProfilePath(os.Getenv("HOME") + "/chainquery")).Stop()

	rand.Seed(time.Now().UnixNano())
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	http.DefaultClient.Timeout = config.GetDefaultClientTimeout()

	if len(os.Args) < 2 { //
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
	case "version":
		println("ALPHA")
	case "test":
		log.Println(`¯\_(ツ)_/¯`)
	case "serve":
		teardown := webServerSetup()
		defer teardown()
		daemon.InitDaemon()

	default:
		log.Errorf("Invalid command: '%s'\n", command)
	}
}

func webServerSetup() func() {
	teardownFuncs := []func(){}

	dbInstance, err := db.Init(config.GetMySQLDSN(), DebugMode)
	if err != nil {
		panic(err) //
	}
	teardownFuncs = append(teardownFuncs, func() { dbInstance.Close() })
	lbrycrdClient, err := lbrycrd.New(config.GetLBRYcrdURL())
	if err != nil {
		panic(err)
	}

	teardownFuncs = append(teardownFuncs, func() { lbrycrdClient.Shutdown() })
	lbrycrd.SetDefaultClient(lbrycrdClient)

	_, err = lbrycrdClient.GetBalance("")
	if err != nil { //
		log.Panicf("Error connecting to lbrycrd: %+v", err)
	}

	return func() {
		for _, f := range teardownFuncs {
			f()
		}
	}
}
