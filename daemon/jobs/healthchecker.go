package jobs

import (
	"time"

	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/chainquery/util"
	"github.com/lbryio/lbry.go/jsonrpc"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

type fileToCheck struct {
	Name    string
	ClaimID string
}

func (f *fileToCheck) Execute() error {
	fileCheck(f.Name, f.ClaimID)
	return nil
}

func (f *fileToCheck) BeforeExecute()    {}
func (f *fileToCheck) AfterExecute()     { logrus.Info(f.ClaimID) }
func (f *fileToCheck) OnError(err error) {}

var workQueue = util.NewQueue()
var client = jsonrpc.NewClient("")
var timeout uint = 60     //seconds
var timeout64 uint64 = 60 //seconds
var running bool

// CheckDHTHealth is job that runs in the background and traverses over claims to check their file status on lbrynet
// via the lbrynet daemon. It stores peers and the associations to claims. It also stores checkpoints for claims and
// peers. For claims it is if they are generally available, for peers it is if their associated files are available
// from them.
func CheckDHTHealth() {
	if !running {
		running = true
		client.SetRPCTimeout(0 * time.Second)
		waitgroup := util.InitWorkers(30, workQueue)
		claims, err := model.ClaimsG(
			qm.Select(model.ClaimColumns.Name, model.ClaimColumns.ClaimID),
			qm.Where(model.ClaimColumns.SDHash+"!=?", ""),
			qm.OrderBy(model.ClaimColumns.Modified+" DESC"),
			qm.Limit(10)).All() //

		if err != nil {
			logrus.Panic(err)
		}
		logrus.Info("Check DHT Health: NrClaims - ", len(claims))
		for i, claim := range claims {
			if i%1 == 0 && i != 0 {
				logrus.Info("Check DHT Health: Processing Claim ", i)
			}
			workQueue <- &fileToCheck{Name: claim.Name, ClaimID: claim.ClaimID}
		}
		close(workQueue)
		waitgroup.Wait()
		logrus.Info("Check DHT Health: Completed DHT Health Check")
		running = false
	}
}

func fileCheck(name string, claimid string) {

	url := name + "#" + claimid
	response, err := client.Resolve(url)
	if err != nil {
		logrus.Warn("Check DHT Health: Unresolvable:->", err)
		return
	}
	if response != nil {
		streamAvailability, err := claimCheckPoint(url, claimid)
		if err != nil {
			logrus.Warn("Check DHT Health: Stream Availability:->", err, " Claimid:", claimid)
			return
		}
		//SDBlob
		if streamAvailability.SDHash != "" {
			peerList, err := client.PeerList(streamAvailability.SDHash, &timeout)
			if err != nil {
				logrus.Warn("Check DHT Health: PeerList:->(", streamAvailability.SDHash, ")", err)
				return
			}

			for _, claimpeer := range *peerList {
				err := setPeer(claimpeer.NodeId)
				if err != nil {
					logrus.Warn("Check DHT Health: Set Peer:", err)
					continue
				}
				err = setPeerClaim(claimpeer.NodeId, claimid)
				if err != nil {
					logrus.Warn("Check DHT Health: Set Peer Claim:", err)
					continue
				}
				err = setPeerClaimCheckPoint(claimpeer.NodeId, claimid)
				if err != nil {
					logrus.Warn("Check DHT Health: Set Peer Claim Checkpoint:", err)
					continue
				}
			}
		} else {
			logrus.Warn("Check DHT Health: Missing SD Hash")
		}
	}
}

func setPeerClaimCheckPoint(nodeid string, claimid string) error {
	pcCheck := &model.PeerClaimCheckpoint{ClaimID: claimid, PeerID: nodeid, Checkpoint: time.Now(), IsAvailable: true}

	err := pcCheck.InsertG()
	if err != nil {
		return err
	}

	return nil
}

func setPeerClaim(nodeid string, claimid string) error {
	peerClaim, _ := model.FindPeerClaimG(nodeid, claimid)
	if peerClaim == nil {
		peerClaim = &model.PeerClaim{PeerID: nodeid, ClaimID: claimid, LastSeen: time.Now()}
		err := peerClaim.InsertG()
		if err != nil {
			return err
		}
	}

	peerClaim.LastSeen = time.Now()
	err := peerClaim.UpdateG()
	if err != nil {
		return err
	}
	return nil
}

func setPeer(nodeid string) error {
	peer, _ := model.FindPeerG(nodeid)
	if peer == nil {
		peer = &model.Peer{NodeID: nodeid}
		err := peer.InsertG()
		if err != nil {
			return err
		}
	}

	return nil
}

func claimCheckPoint(url string, claimid string) (*jsonrpc.StreamAvailabilityResponse, error) {
	result, err := client.StreamAvailability(url, &timeout64, &timeout64)
	if err != nil {
		return nil, err
	}
	println("url ", url)
	claimCheckpoint := model.ClaimCheckpoint{}
	claimCheckpoint.ClaimID = claimid
	claimCheckpoint.Checkpoint = time.Now()
	claimCheckpoint.IsAvailable = result.IsAvailable
	claimCheckpoint.HeadAvailable = result.HeadBlobAvailability.IsAvailable
	claimCheckpoint.SDAvailable = result.SDBlobAvailability.IsAvailable

	err = claimCheckpoint.InsertG()
	if err != nil {
		return nil, err
	}

	return result, nil
}
