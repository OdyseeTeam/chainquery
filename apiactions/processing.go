package apiactions

import (
	"net/http"

	"github.com/lbryio/chainquery/auth"
	"github.com/lbryio/chainquery/daemon/jobs"
	"github.com/lbryio/chainquery/daemon/processing"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"

	"github.com/lbryio/lbry.go/v2/extras/api"
	"github.com/lbryio/lbry.go/v2/extras/errors"

	v "github.com/lbryio/ozzo-validation"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

// ProcessBlocks processed a specific block or range of blocks if authorized.
func ProcessBlocks(r *http.Request) api.Response {
	params := struct {
		Block *uint64
		From  *uint64
		To    *uint64
		Key   string
	}{}

	err := api.FormValues(r, &params, []*v.FieldRules{
		v.Field(&params.Block),
		v.Field(&params.From),
		v.Field(&params.To),
		v.Field(&params.Key),
	})
	if err != nil {
		return api.Response{Error: err, Status: http.StatusBadRequest}
	}

	if !auth.IsAuthorized(params.Key) {
		return api.Response{Error: errors.Err("not authorized"), Status: http.StatusUnauthorized}
	}

	if params.Block != nil {
		err = processBlocks(params.Block, nil, nil)
	} else if params.To != nil {
		err = processBlocks(nil, params.From, params.To)
	} else {
		err = processBlocks(nil, params.From, nil)
	}

	if err != nil {
		return api.Response{Error: err}
	}

	return api.Response{Data: "OK"}

}

func processBlocks(block, from, to *uint64) error {
	if block != nil {
		from = block
		end := *block
		to = &end
	} else {
		if from == nil {
			start := uint64(0)
			from = &start
		}
		if to == nil {
			currHeight, err := lbrycrd.GetBlockCount()
			if err != nil {
				return errors.Err(err)
			}
			to = currHeight
		}
	}

	for *from <= *to {
		processing.RunBlockProcessing(nil, *from)
		*from++
	}
	return nil
}

// SyncName syncs the claims for a give name with the lbrycrd claimtrie ( effective amount, valid at height, and bidstate).
func SyncName(r *http.Request) api.Response {

	params := struct {
		Name string
		Key  string
	}{}

	err := api.FormValues(r, &params, []*v.FieldRules{
		v.Field(&params.Name),
		v.Field(&params.Key),
	})
	if err != nil {
		return api.Response{Error: err, Status: http.StatusBadRequest}
	}

	if !auth.IsAuthorized(params.Key) {
		return api.Response{Error: errors.Err("not authorized"), Status: http.StatusUnauthorized}
	}

	claims, err := model.Claims(qm.Where(model.ClaimColumns.Name+"=?", params.Name), qm.Limit(1)).AllG()
	if err != nil {
		return api.Response{Error: errors.Err(err)}
	}

	count, err := lbrycrd.GetBlockCount()
	if err != nil {
		logrus.Error("ClaimTrieSyncAsync: Error getting block height", err)
		return api.Response{Error: errors.Err(err)}
	}

	err = jobs.SyncClaims(claims)
	if err != nil {
		return api.Response{Error: errors.Err(err)}
	}

	err = jobs.SetControllingClaimForNames(claims, *count)
	if err != nil {
		return api.Response{Error: errors.Err(err)}
	}

	return api.Response{Data: "ok"}

}
