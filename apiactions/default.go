package apiactions

import (
	"net/http"
	"os/exec"
	"time"

	"github.com/lbryio/chainquery/db"
	. "github.com/lbryio/lbry.go/api"
	"github.com/lbryio/lbry.go/travis"
	v "github.com/lbryio/ozzo-validation"

	"github.com/lbryio/lbry.go/errors"
	"github.com/sirupsen/logrus"
)

func ChainQueryStatusAction(r *http.Request) Response {

	status, err := db.GetTableStatus()
	if err != nil {
		return Response{Error: err}
	}
	return Response{Data: status}
}

func AddressSummaryAction(r *http.Request) Response {
	params := struct {
		Address string
	}{}
	err := FormValues(r, &params, []*v.FieldRules{
		v.Field(&params.Address, v.Required),
	})
	if err != nil {
		return Response{Error: err, Status: http.StatusBadRequest}
	}
	summary, err := db.GetAddressSummary(params.Address)
	if err != nil {
		return Response{Error: err, Status: http.StatusInternalServerError}
	}
	return Response{Data: summary}
}

func SQLQueryAction(r *http.Request) Response {
	query := r.FormValue("query")
	result, err := db.APIQuery(query)
	if err != nil {
		return Response{Error: err, Status: http.StatusBadRequest}
	}
	return Response{Data: result}
}

func IndexAction(r *http.Request) Response {
	return Response{Data: "Hello World!"}
}

var AutoUpdateCommand = ""

// SelfUpdateAction takes a travis webhook for a successful deployment and runs an environment script to self update.
func AutoUpdateAction(r *http.Request) Response {
	err := travis.ValidateSignature(r)
	logrus.Info(err)
	if err != nil {
		return Response{Error: err, Status: http.StatusBadRequest}
	}

	webHook, err := travis.NewFromRequest(r)
	if err != nil {
		return Response{Error: err}
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

	return Response{Data: "ok"}
}
