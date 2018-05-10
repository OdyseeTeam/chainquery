package apiactions

import (
	"net/http"
	"os/exec"
	"time"

	"github.com/lbryio/chainquery/db"
	"github.com/lbryio/lbry.go/api"
	"github.com/lbryio/lbry.go/errors"
	"github.com/lbryio/lbry.go/travis"
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

// SQLQueryAction returns an array of structured data matching the queried results.
func SQLQueryAction(r *http.Request) api.Response {
	query := r.FormValue("query")
	result, err := db.APIQuery(query)
	if err != nil {
		return api.Response{Error: err, Status: http.StatusBadRequest}
	}
	return api.Response{Data: result}
}

// IndexAction returns Hello World!
func IndexAction(r *http.Request) api.Response {
	return api.Response{Data: "Hello World!"}
}

var AutoUpdateCommand = ""

// AutoUpdateAction takes a travis webhook for a successful deployment and runs an environment script to self update.
func AutoUpdateAction(r *http.Request) api.Response {
	err := travis.ValidateSignature(r)
	logrus.Info(err)
	if err != nil {
		return api.Response{Error: err, Status: http.StatusBadRequest}
	}

	webHook, err := travis.NewFromRequest(r)
	if err != nil {
		return api.Response{Error: err}
	}

	if webHook.Status == 0 { // webHook.ShouldDeploy() doesn't work for chainquery autoupdate.
		if AutoUpdateCommand == "" {
			logrus.Warnln("self-update triggered, but no self-update command configured")
		} else {
			logrus.Info("self-updating")
			// run self-update asynchronously
			go func() {
				time.Sleep(1 * time.Second) // leave time for handler to send response
				cmd := exec.Command(AutoUpdateCommand)
				out, err := cmd.Output()
				if err != nil {
					errMsg := "self-update error: " + errors.FullTrace(err) + "\nStdout: " + string(out)
					if exitErr, ok := err.(*exec.ExitError); ok {
						errMsg = errMsg + "\nStderr: " + string(exitErr.Stderr)
					}
					logrus.Errorln(errMsg)
				}
			}()
		}
	}

	return api.Response{Data: "ok"}
}
