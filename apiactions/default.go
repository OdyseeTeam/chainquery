package apiactions

import (
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/lbryio/chainquery/db"
	"github.com/lbryio/chainquery/meta"
	"github.com/lbryio/lbry.go/extras/api"
	"github.com/lbryio/lbry.go/extras/errors"
	"github.com/lbryio/lbry.go/extras/travis"

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

//MaxSQLAPITimeout sets a timeout, in seconds, on queries placed against the SQL API.
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

// AutoUpdateCommand is the path of the shell script to run in the environment chainquery is installed on. It should
// stop the service, download and replace the new binary from https://github.com/lbryio/chainquery/releases, start the
// service.
var AutoUpdateCommand = ""

// AutoUpdateAction takes a travis webhook for a successful deployment and runs an environment script to self update.
func AutoUpdateAction(r *http.Request) api.Response {
	err := travis.ValidateSignature(false, r)
	if err != nil {
		logrus.Info(err)
		return api.Response{Error: err, Status: http.StatusBadRequest}
	}

	webHook, err := travis.NewFromRequest(r)
	if err != nil {
		return api.Response{Error: err}
	}

	if webHook.Commit == meta.GetVersion() {
		return api.Response{Data: "same commit version, skipping automatic update."}
	}

	shouldUpdate := webHook.Status == 0 && !webHook.PullRequest && webHook.Tag != "" && webHook.Tag != meta.GetVersion()
	if shouldUpdate { // webHook.ShouldDeploy() doesn't work for chainquery autoupdate.
		if AutoUpdateCommand == "" {
			err := errors.Base("auto-update triggered, but no auto-update command configured")
			logrus.Error(err)
			return api.Response{Error: err}
		}
		logrus.Info("chainquery is auto-updating...prepare for shutdown")
		// run auto-update asynchronously
		go func() {
			time.Sleep(1 * time.Second) // leave time for handler to send response
			cmd := exec.Command(AutoUpdateCommand)
			out, err := cmd.Output()
			if err != nil {
				errMsg := "auto-update error: " + errors.FullTrace(err) + "\nStdout: " + string(out)
				if exitErr, ok := err.(*exec.ExitError); ok {
					errMsg = errMsg + "\nStderr: " + string(exitErr.Stderr)
				}
				logrus.Errorln(errMsg)
			}
		}()
		return api.Response{Data: "Successful launch of auto update"}

	}
	message := "Auto-Update should not be deployed for one of the following:" +
		" CI Status-" + webHook.StatusMessage +
		", IsPullRequest-" + strconv.FormatBool(webHook.PullRequest) +
		", TagName-" + webHook.Tag
	return api.Response{Data: message}
}
