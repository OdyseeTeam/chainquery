package apiactions

import (
	"net/http"

	"github.com/lbryio/chainquery/auth"
	"github.com/lbryio/chainquery/daemon/processing"
	"github.com/lbryio/chainquery/lbrycrd"

	"github.com/lbryio/lbry.go/api"
	"github.com/lbryio/lbry.go/errors"

	v "github.com/lbryio/ozzo-validation"
)

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

	if (params.Block != nil && *params.Block < 0) || (params.From != nil && *params.From < 0) || (params.To != nil && *params.To < 0) {
		return api.Response{Error: errors.Err("a positive value must be passed"), Status: http.StatusBadRequest}
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
		processing.RunBlockProcessing(*from)
		*from++
	}
	return nil
}
