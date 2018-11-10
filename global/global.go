package global

import "time"

// DaemonSettings is a struct for holding the different settings of the daemon.
type DaemonSettings struct {
	DaemonMode      int
	ProcessingDelay time.Duration
	DaemonDelay     time.Duration
	IsReIndex       bool
}

//BlockChainName is the name of the blockchain. It is used to decode protobuf claims.
const BlockChainName = "lbrycrd_main"
