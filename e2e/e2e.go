package e2e

import (
	"database/sql"
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
	jobs.ClaimTrieSync()
	jobs.CertificateSync()
	exitOnErr(jobs.SyncClaimCntInChannel())
	time.Sleep(2 * time.Second)
	exitOnErr(checkCertValid([]string{"claim1", "claim2", "claim3"}))
	testClaimCount()
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
		hash, err := lbrycrd.GenerateBlocks(int64(blocks[0]))
		exitOnErr(err)
		_, err = checkForBlock(hash[len(hash)-1])
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

func checkForBlock(blockHash string) (bool, error) {
	start := time.Now()
	ticker := time.NewTicker(500 * time.Millisecond)
	for t := range ticker.C {
		tx, err := model.Blocks(model.BlockWhere.Hash.EQ(blockHash)).OneG()
		if err != nil && err != sql.ErrNoRows {
			return false, errors.Err(err)
		}
		if tx != nil {
			ticker.Stop()
			return true, nil
		}
		if start.Add(5 * time.Minute).Before(t) {
			return false, nil
		}
	}
	return false, nil
}

func testClaimCount() {
	channel, err := model.Claims(model.ClaimWhere.Name.EQ("@MyChannel")).OneG()
	exitOnErr(err)
	if channel == nil {
		exit(1, errors.Err("Could not find channel @MyChannel"))
	}
	if channel.ClaimCount != 3 {
		exit(1, errors.Err("@MyChannel only has %d claims and should have 3", channel.ClaimCount))
	}
}
