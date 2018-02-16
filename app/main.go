package main

//go:generate go-bindata -o migration/bindata.go -pkg migration -ignore bindata.go migration/

import (
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time" //

	"github.com/lbryio/chainquery/app/db"
	"github.com/lbryio/chainquery/app/env"
	"github.com/lbryio/lbry.go/lbrycrd"

	log "github.com/sirupsen/logrus"
)

var DebugMode = false

func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	http.DefaultClient.Timeout = 20 * time.Second

	if len(os.Args) < 2 {
		log.Errorf("Usage: %s COMMAND", os.Args[0])
		return
	}

	DebugMode, err := strconv.ParseBool(os.Getenv("DEBUGGING"))
	if err != nil {
		panic(err)
	}
	if DebugMode {
		log.SetLevel(log.DebugLevel)
	}

	command := os.Args[1]
	switch command {
	case "test":
		log.Println(`¯\_(ツ)_/¯`)
	case "serve":
		conf, err := env.NewWithEnvVars()
		if err != nil {
			panic(err)
		}
		teardown := webServerSetup(conf)
		defer teardown()
	default:
		log.Errorf("Invalid command: '%s'\n", command)
	}
}

func webServerSetup(conf *env.Config) func() {
	teardownFuncs := []func(){}

	dbInstance, err := db.Init(conf.MysqlDsn, DebugMode)
	if err != nil {
		panic(err)
	}
	teardownFuncs = append(teardownFuncs, func() { dbInstance.Close() })
	println(conf.LbrycrdURL)
	if conf.LbrycrdURL != "" {
		lbrycrdClient, err := lbrycrd.New(conf.LbrycrdURL)
		if err != nil {
			panic(err)
		}

		teardownFuncs = append(teardownFuncs, func() { lbrycrdClient.Shutdown() })
		lbrycrd.SetDefaultClient(lbrycrdClient)

		_, err = lbrycrdClient.GetBalance("")
		if err != nil {
			log.Panicf("Error connecting to lbrycrd: %+v", err)
		} //
		print("Connected successfully to lbrycrd")
	}

	return func() {
		for _, f := range teardownFuncs {
			f()
		}
	}
}
