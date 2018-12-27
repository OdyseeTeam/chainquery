package apiactions

import (
	"net/http"

	"github.com/lbryio/chainquery/auth"
	"github.com/lbryio/chainquery/daemon/jobs"

	"github.com/lbryio/lbry.go/api"
	"github.com/lbryio/lbry.go/errors"

	v "github.com/lbryio/ozzo-validation"
)

//SyncAddressBalance will synchronize the balances for all addresses in chainquery.
func SyncAddressBalance(r *http.Request) api.Response {
	params := struct {
		Key string
	}{}

	err := api.FormValues(r, &params, []*v.FieldRules{
		v.Field(&params.Key),
	})
	if err != nil {
		return api.Response{Error: err, Status: http.StatusBadRequest}
	}

	if !auth.IsAuthorized(params.Key) {
		return api.Response{Error: errors.Err("not authorized"), Status: http.StatusUnauthorized}
	}

	rowsAffected, err := jobs.SyncAddressBalances()
	if err != nil {
		return api.Response{Error: err}
	}

	return api.Response{Data: rowsAffected}

}

//SyncTransactionValue will synchronize the value of all transactions in chainquery.
func SyncTransactionValue(r *http.Request) api.Response {
	params := struct {
		Key string
	}{}

	err := api.FormValues(r, &params, []*v.FieldRules{
		v.Field(&params.Key),
	})
	if err != nil {
		return api.Response{Error: err, Status: http.StatusBadRequest}
	}

	if !auth.IsAuthorized(params.Key) {
		return api.Response{Error: errors.Err("not authorized"), Status: http.StatusUnauthorized}
	}

	rowsAffected, err := jobs.SyncTransactionValue()
	if err != nil {
		return api.Response{Error: err}
	}

	return api.Response{Data: rowsAffected}

}

// ValidateChainData validates a range of blocks ensure that the block,Txs, and the same number of outputs,inputs exist.
//If a difference in data is identified it will return an array identifying where there are differences.
func ValidateChainData(r *http.Request) api.Response {
	params := struct {
		From uint64
		To   *uint64
		Key  string
	}{}

	err := api.FormValues(r, &params, []*v.FieldRules{
		v.Field(&params.From, v.Required),
		v.Field(&params.To),
		v.Field(&params.Key),
	})
	if err != nil {
		return api.Response{Error: err, Status: http.StatusBadRequest}
	}

	if !auth.IsAuthorized(params.Key) {
		return api.Response{Error: errors.Err("not authorized"), Status: http.StatusUnauthorized}
	}

	var missing []jobs.BlockData
	if params.To != nil {
		missing, err = jobs.ValidateChainRange(&params.From, params.To)
	} else {
		missing, err = jobs.ValidateChainRange(&params.From, nil)
	}

	if err != nil {
		return api.Response{Error: err}
	}

	return api.Response{Data: missing}
}
