package e2e

import (
	"os"
	"time"

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
	logrus.Info(lbrycrd.ClaimName("chainquery", "636861696e717565727920697320617765736f6d6521", 0.01))
	//Legit Claims
	logrus.Info(lbrycrd.ClaimName("example001", "7b22736f7572636573223a207b226c6272795f73645f68617368223a2022313938613661393231356435363539316235633934343634353363356361626238343236646634383335316365393866353731363562623336323337663535366263613433336636633038653635393866376234663835653733656136366638227d7d", 0.01))
	logrus.Info(lbrycrd.ClaimName("example002", "080110011afb03080112b303080410011a26446576696c204d6179204372792048442028772f436f6d6d656e746172792920506172742035229a02446576696c204d61792043727920506c61797468726f7567680a436f6e736f6c65202d205053332028446576696c204d61792043727920484420436f6c6c656374696f6e290a47616d65706c6179202d20687474703a2f2f7777772e54686542726f74686572686f6f646f6647616d696e672e636f6d0a506c617965727320312026203220436f6d6d656e74617279202d2057696c6c69616d204d6f72726973202620457567656e65204d6f727269730a54776974746572202d2068747470733a2f2f747769747465722e636f6d2f54424f476d6f72726973313131330a46616365626f6f6b202d2068747470733a2f2f7777772e66616365626f6f6b2e636f6d2f54424f476d6f72726973313131330a5355425343524942452a1a54686542726f74686572686f6f646f6647616d696e672e636f6d321c436f7079726967687465642028436f6e7461637420417574686f722938004a28687474703a2f2f6265726b2e6e696e6a612f7468756d626e61696c732f534e4e4c4162595331425152005a001a41080110011a30f6b79604c847c80821a2bb18a2a74c46180aaa8ae8a7731e321b9ca445e278cb81352feec9f56a9f85808dec8752ced92209766964656f2f6d70342a5c080110031a405cf7e6bded265492bf8a55434dacbfca358e5c2d1270ad8784ff4c0f3029e61af62056d7439f2296bba66edf69c52502d2c196c84d3f79e59f73911dd77dd14e2214b32a8f12c3a2e74eaad01a5ff9647aa1aa4038e0", 0.01))
	logrus.Info(lbrycrd.ClaimName("example003", "080110011adc010801129401080410011a0d57686174206973204c4252593f223057686174206973204c4252593f20416e20696e74726f64756374696f6e207769746820416c6578205461626172726f6b2a0c53616d75656c20427279616e32084c42525920496e6338004a2f68747470733a2f2f73332e616d617a6f6e6177732e636f6d2f66696c65732e6c6272792e696f2f6c6f676f2e706e6752005a001a41080110011a30d5169241150022f996fa7cd6a9a1c421937276a3275eb912790bd07ba7aec1fac5fd45431d226b8fb402691e79aeb24b2209766964656f2f6d7034", 0.01))
	//Channels
	logrus.Info(lbrycrd.ClaimName("channel001", "08011002225e0801100322583056301006072a8648ce3d020106052b8104000a0342000422ae63a64fd2cff5e698072c1a2af8117cc734b12458321946aec08081b78ce5498e9b4325b6d0352a7ab1dfe1b951a75f290f1321b26901886bb8a2fee59c0f", 0.01))
	logrus.Info(lbrycrd.ClaimName("channel002", "08011002225e0801100322583056301006072a8648ce3d020106052b8104000a0342000490b9d5049a72bdf7ce6b6e11f9108a3b92fcaf431d29e6040f36a2786c95e4a42a81a859fd80951f1f459113c4d781cb3647222ce0cf0ba2d117d362e823510e", 0.01))
	logrus.Info(lbrycrd.GenerateBlocks(10))
	createChannelWithClaims()
	logrus.Info(lbrycrd.GenerateBlocks(1))
	time.Sleep(15 * time.Second)
	jobs.ClaimTrieSync()
	jobs.CertificateSync()
	time.Sleep(2 * time.Second)
	exitOnErr(checkCertValid([]string{"claim1", "claim2", "claim3"}))
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

	claim1, err := newClaim("title1", "description2")
	if err != nil {
		exit(1, err)
	}

	claim2, err := newClaim("title1", "description2")
	if err != nil {
		exit(1, err)
	}
	claim3, err := newClaim("title1", "description2")
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
	logrus.Error(err)
	os.Exit(code)
}

func exitOnErr(err error) {
	if err != nil {
		exit(1, err)
	}
}
