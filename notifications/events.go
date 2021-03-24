package notifications

import (
	"net/url"

	"github.com/spf13/cast"
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
}

// ClaimEvent event to notify subscribers of a new claim thats been published
func ClaimEvent(claimID, name, title, txID, channeClaimID, source string) {
	values := url.Values{}
	values.Add("claim_id", claimID)
	values.Add("name", name)
	values.Add("title", title)
	values.Add("tx_id", txID)
	values.Add("source", source)
	values.Add("channel_claim_id", channeClaimID)
	go Notify(newClaim, values)
}
