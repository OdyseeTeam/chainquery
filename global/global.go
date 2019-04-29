package global

import "time"

// DaemonSettings is a struct for holding the different settings of the daemon.
type DaemonSettings struct {
	DaemonMode      int
	ProcessingDelay time.Duration
	DaemonDelay     time.Duration
	IsReIndex       bool
}

// BlockChainName is the name of the blockchain. It is used to decode protobuf claims.
var BlockChainName = "lbrycrd_main"

const (
	//StreamClaimType is a reference to claim table type column - stream claims
	StreamClaimType = "stream"
	//ChannelClaimType is a reference to claim table type column - channel claims
	ChannelClaimType = "channel"
	//ClaimListClaimType is a reference to claim table type column - list of claims
	ClaimListClaimType = "claimlist"
	//ClaimReferenceClaimType is a reference to claim table type column - reference to another claim
	ClaimReferenceClaimType = "claimreference"
)
