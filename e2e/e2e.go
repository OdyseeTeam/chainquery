package e2e

import (
	"time"

	"github.com/lbryio/chainquery/daemon"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/sirupsen/logrus"
)

// StartE2ETesting launches the test suite
func StartE2ETesting() {
	go daemon.DoYourThing()
	logrus.Info("Started E2E tests")
	logrus.Info(lbrycrd.ClaimName("chainquery", "636861696e717565727920697320617765736f6d6521", 0.01))
	logrus.Info(lbrycrd.GenerateBlocks(1))
	time.Sleep(10 * time.Second)
	daemon.ShutdownDaemon()
}
