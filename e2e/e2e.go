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
	logrus.Info(lbrycrd.ClaimName("example010", "080110011adc010801129401080410011a0d57686174206973204c4252593f223057686174206973204c4252593f20416e20696e74726f64756374696f6e207769746820416c6578205461626172726f6b2a0c53616d75656c20427279616e32084c42525920496e6338004a2f68747470733a2f2f73332e616d617a6f6e6177732e636f6d2f66696c65732e6c6272792e696f2f6c6f676f2e706e6752005a001a41080110011a30d5169241150022f996fa7cd6a9a1c421937276a3275eb912790bd07ba7aec1fac5fd45431d226b8fb402691e79aeb24b2209766964656f2f6d7034", 0.01))
	logrus.Info(lbrycrd.GenerateBlocks(1))
	time.Sleep(10 * time.Second)
	daemon.ShutdownDaemon()
}
