package apiactions

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lbryio/chainquery/db"
	"github.com/lbryio/lbry.go/v2/extras/api"
	"github.com/lbryio/lbry.go/v2/extras/errors"

	v "github.com/lbryio/ozzo-validation"
	"github.com/sirupsen/logrus"
)

// ChainQueryStatusAction returns status information of the chainquery application. Currently, tables' name and size.
func ChainQueryStatusAction(r *http.Request) api.Response {

	status, err := db.GetTableStatus()
	if err != nil {
		return api.Response{Error: err}
	}
	return api.Response{Data: status}
}

// AddressSummaryAction returns address details: received, spent, balance
func AddressSummaryAction(r *http.Request) api.Response {
	params := struct {
		Address string
	}{}
	err := api.FormValues(r, &params, []*v.FieldRules{
		v.Field(&params.Address, v.Required),
	})
	if err != nil {
		return api.Response{Error: err, Status: http.StatusBadRequest}
	}
	summary, err := db.GetAddressSummary(params.Address)
	if err != nil {
		return api.Response{Error: err, Status: http.StatusInternalServerError}
	}
	return api.Response{Data: summary}
}

// MaxSQLAPITimeout sets a timeout, in seconds, on queries placed against the SQL API.
var MaxSQLAPITimeout int

// SQLQueryAction returns an array of structured data matching the queried results.
func SQLQueryAction(r *http.Request) api.Response {
	query := r.FormValue("query")
	query = strings.Replace(strings.ToLower(query), "select ", fmt.Sprintf("select /*+ MAX_EXECUTION_TIME(%d) */", MaxSQLAPITimeout*1000), 1)
	logrus.Debugf("Query: %s", query)
	start := time.Now()
	result, err := db.APIQuery(query)
	if err != nil {
		return api.Response{Error: err, Status: http.StatusBadRequest}
	}
	if time.Since(start) > time.Duration(MaxSQLAPITimeout)*time.Second {
		return api.Response{Error: errors.Err(``+
			`Queries must take less than %d seconds or they are cancelled, to give everyone a chance to use the API since this is a `+
			`public API. If you have a query you really want/need please create an issue in the chainquery repo. We are always happy `+
			`to add indices to make queries faster.`, MaxSQLAPITimeout), Status: http.StatusBadRequest}
	}
	return api.Response{Data: result}
}

// IndexAction returns Hello World!
func IndexAction(r *http.Request) api.Response {
	return api.Response{Data: "Hello World!"}
}
