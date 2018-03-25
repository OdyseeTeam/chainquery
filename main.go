package main

//go:generate go-bindata -o migration/bindata.go -pkg migration -ignore bindata.go migration/

import (
	"flag"
	"math/rand"
	"net/http"
	"os"
	"runtime/pprof"
	"strconv"
	"time"

	"github.com/lbryio/chainquery/daemon"
	"github.com/lbryio/chainquery/db"
	"github.com/lbryio/chainquery/env"
	"github.com/lbryio/chainquery/lbrycrd"

	log "github.com/sirupsen/logrus"
)

var DebugMode bool
var cpuprofile = flag.String("cpuprofile", "./chainquery.prof", "write cpu profile to file")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		log.Info("Starting Profiler")
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	rand.Seed(time.Now().UnixNano())
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	http.DefaultClient.Timeout = 20 * time.Second
	daemon.ProcessingMode = daemon.BEASTMODE // TODO: Should be configurable

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
		daemon.InitDaemon()

	default:
		log.Errorf("Invalid command: '%s'\n", command)
	}
}

func webServerSetup(conf *env.Config) func() {
	teardownFuncs := []func(){}

	dbInstance, err := db.Init(conf.MysqlDsn, DebugMode)
	if err != nil {
		panic(err) //
	}
	teardownFuncs = append(teardownFuncs, func() { dbInstance.Close() })
	if conf.LbrycrdURL != "" {
		lbrycrdClient, err := lbrycrd.New(conf.LbrycrdURL)
		if err != nil {
			panic(err)
		}

		teardownFuncs = append(teardownFuncs, func() { lbrycrdClient.Shutdown() })
		lbrycrd.SetDefaultClient(lbrycrdClient)

		_, err = lbrycrdClient.GetBalance("")
		if err != nil { //
			log.Panicf("Error connecting to lbrycrd: %+v", err)
		} //
		println("Connected successfully to lbrycrd")
	}

	return func() {
		for _, f := range teardownFuncs {
			f()
		}
	}
}
