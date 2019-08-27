package e2e

import (
	"os"
	"time"

	"github.com/lbryio/chainquery/model"
	"github.com/volatiletech/sqlboiler/queries/qm"

	"github.com/lbryio/lbry.go/extras/errors"

	"github.com/lbryio/chainquery/daemon"
	"github.com/lbryio/chainquery/daemon/jobs"
	"github.com/lbryio/chainquery/lbrycrd"

	"github.com/sirupsen/logrus"
)

const claimAmount = 0.5

// StartE2ETesting launches the test suite
func StartE2ETesting() {
	go daemon.DoYourThing()
	logrus.Info("Started E2E tests")
	testBasicClaimCreation()
	increment(10)
	createChannelWithClaims()
	increment(1)
	time.Sleep(15 * time.Second)
	jobs.ClaimTrieSync()
	jobs.CertificateSync()
	time.Sleep(2 * time.Second)
	exitOnErr(checkCertValid([]string{"claim1", "claim2", "claim3"}))
	testImageMetadata()
	testVideoMetaData()
	daemon.ShutdownDaemon()
}

func createChannelWithClaims() {

	channel, key, err := createChannel("@MyChannel")
	if err != nil {
		exit(1, err)
	}

	logrus.Info(lbrycrd.GenerateBlocks(1))

	claimResponse, err := lbrycrd.GetClaimsForName("@MyChannel")
	if err != nil {
		exit(1, err)
	}

	claim1, err := newStreamClaim("title1", "description2")
	if err != nil {
		exit(1, err)
	}

	claim2, err := newStreamClaim("title1", "description2")
	if err != nil {
		exit(1, err)
	}
	claim3, err := newStreamClaim("title1", "description2")
	if err != nil {
		exit(1, err)
	}

	rawTx, err := getEmptyTx(claimAmount * 3)
	if err != nil {
		exit(1, err)
	}

	exitOnErr(signClaim(rawTx, *key, claim1, channel, claimResponse.Claims[0].ClaimID))
	exitOnErr(signClaim(rawTx, *key, claim2, channel, claimResponse.Claims[0].ClaimID))
	exitOnErr(signClaim(rawTx, *key, claim3, channel, claimResponse.Claims[0].ClaimID))

	exitOnErr(addClaimToTx(rawTx, claim1, "claim1"))
	exitOnErr(addClaimToTx(rawTx, claim2, "claim2"))
	exitOnErr(addClaimToTx(rawTx, claim3, "claim3"))

	chainHash, err := signTxAndSend(rawTx)
	if err != nil {
		exit(1, err)
	}

	logrus.Info("Tx:", chainHash.String())

}

func exit(code int, err error) {
	logrus.Error(err, "\n", errors.FullTrace(err))
	os.Exit(code)
}

func exitOnErr(err error) {
	if err != nil {
		exit(1, errors.Err(err))
	}
}

func increment(blocks ...int) {
	if len(blocks) > 0 {
		_, err := lbrycrd.GenerateBlocks(int64(blocks[0]))
		exitOnErr(err)
	}
}

func test(run func(), check func() error, blocks uint) {
	height, err := lbrycrd.GetBlockCount()
	exitOnErr(errors.Err(err))
	run()
	increment(int(blocks + 1))
	//wait x blocks
	reached := make(chan bool)
	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		for range ticker.C {
			block, err := model.Blocks(qm.OrderBy("height DESC"), qm.Limit(1)).OneG()
			exitOnErr(errors.Err(err))
			if (block.Height - *height) >= uint64(blocks+1) {
				logrus.Infof("Block %d reached from %d", block.Height, *height)
				reached <- true
				ticker.Stop()
			}
		}
	}()
	<-reached
	exitOnErr(check())
}

func verify(test bool, message string) {
	var err error
	if !test {
		err = errors.Err(message)
	}
	exitOnErr(err)
}
