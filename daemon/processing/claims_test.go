package processing

import (
	"github.com/lbryio/chainquery/model"
	"testing"
)

func TestGetClaimIDFromOutput(t *testing.T) {
	goodClaimID := "1318a78e018366a5ee162952872cd91c64aca128"
	output := model.Output{}
	output.TransactionHash = "a6a43bc516b601490853433968ee8147a9a8d4a6ed36beb81ce833d829c0bcd1"
	output.Vout = 1

	claimID := getClaimIdFromOutput(&output)
	if claimID != goodClaimID {
		t.Error("Expected ", goodClaimID, " got ", claimID)
	}

}
