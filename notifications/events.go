package notifications

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/chainquery/sockety"

	"github.com/lbryio/lbry.go/v2/extras/jsonrpc"
	c "github.com/lbryio/lbry.go/v2/schema/stake"
	"github.com/lbryio/sockety/socketyapi"
	
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

const payment = "payment"
const newClaim = "new_claim"

// PaymentEvent event to notify subscribers of a payment transaction
func PaymentEvent(lbc float64, address, txid string, vout uint) {
	values := url.Values{}
	values.Add("lbc", cast.ToString(lbc))
	values.Add("tx_id", txid)
	values.Add("vout", cast.ToString(vout))
	values.Add("address", address)
	go Notify(payment, values)
	go sockety.SendNotification(socketyapi.SendNotificationArgs{
		Service: socketyapi.BlockChain,
		Type:    "payments",
		IDs:     []string{"payments", address, strconv.Itoa(int(lbc * 0.001))},
		Data:    map[string]interface{}{"lbc": lbc, "address": address, "txid": txid, "vout": vout},
	})
}

// ClaimEvent event to notify subscribers of a new claim thats been published
func ClaimEvent(claim *model.Claim, tx model.Transaction, claimData *c.StakeHelper) {
	values := url.Values{}
	values.Add("claim_id", claim.ClaimID)
	values.Add("name", claim.Name)
	if !claim.Type.IsZero() {
		values.Add("type", claim.Type.String)
	}
	if !claim.Title.IsZero() {
		values.Add("title", claim.Title.String)
	}
	if !claim.Description.IsZero() {
		values.Add("description", claim.Description.String)
	}
	if !claim.ThumbnailURL.IsZero() {
		values.Add("thumbnail_url", claim.ThumbnailURL.String)
	}
	if !claim.ReleaseTime.IsZero() {
		values.Add("release_time", strconv.Itoa(int(claim.ReleaseTime.Uint64)))
	}
	if !claim.SDHash.IsZero() {
		values.Add("sd_hash", claim.SDHash.String)
	}
	values.Add("tx_id", tx.Hash)
	if !claim.SourceHash.IsZero() {
		values.Add("source", claim.SourceHash.String)
	}
	if !claim.PublisherID.IsZero() {
		values.Add("channel_claim_id", claim.PublisherID.String)
	}

	isProtected := false
	for _, t := range claimData.Claim.GetTags() {
		if strings.Contains(t, string(jsonrpc.ProtectedContentTag)) {
			isProtected = true
			break
		}
	}
	values.Add("is_protected", strconv.FormatBool(isProtected))
	signingChannel, err := model.Claims(qm.Where("claim_id=?", claim.PublisherID.String)).OneG()
	if err != nil {
		log.Errorf("failed to get signing channel for claim %s: %v", claim.ClaimID, err)
	}
	if signingChannel != nil {
		values.Add("channel_name", signingChannel.Name)
		if !signingChannel.ThumbnailURL.IsZero() {
			values.Add("channel_thumbnail_url", signingChannel.ThumbnailURL.String)
		}
	}

	go Notify(newClaim, values)
}
