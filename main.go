package main

//go:generate go-bindata -o migration/bindata.go -pkg migration -ignore bindata.go migration/

import (
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/lbryio/chainquery/daemon"
	"github.com/lbryio/chainquery/db"
	"github.com/lbryio/chainquery/env"
	"github.com/lbryio/chainquery/lbrycrd"
	log "github.com/sirupsen/logrus"
)

var DebugMode = false

func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})

	DebugMode, err := strconv.ParseBool(os.Getenv("DEBUGGING"))
	if err != nil {
		panic(err)
	}
	if DebugMode {
		log.SetLevel(log.DebugLevel)
	}

	conf, err := env.NewWithEnvVars()
	if err != nil {
		panic(err)
	}

	dbInstance, err := db.Init(conf.MysqlDsn, DebugMode)
	if err != nil {
		panic(err)
	}
	defer dbInstance.Close()

	if conf.LbrycrdURL != "" {
		lbrycrdClient, err := lbrycrd.New(conf.LbrycrdURL)
		if err != nil {
			panic(err)
		}
		defer lbrycrdClient.Shutdown()

		lbrycrd.SetDefaultClient(lbrycrdClient)

		_, err = lbrycrdClient.GetBalance("")
		if err != nil {
			log.Panicf("Error connecting to lbrycrd: %+v", err)
		}
		log.Println("Connected successfully to lbrycrd")
	}

	daemon.InitDaemon()
}
