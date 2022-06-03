/*
 * Chain Query
 *
 * The LBRY blockchain is read into SQL where important structured information can be extracted through the Chain Query API.
 *
 * API version: 0.1.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package swagger

import (
	"net/http"
	"strconv"

	"github.com/lbryio/chainquery/config"
	"github.com/lbryio/chainquery/db"
	sw "github.com/lbryio/chainquery/swagger/apiserver/go"
	"github.com/lbryio/lbry.go/v2/extras/api"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

func InitApiServer(hostAndPort string) {
	logrus.Info("API Server started")
	hs := make(map[string]string)
	hs["Access-Control-Allow-Methods"] = "GET, POST, OPTIONS"
	hs["Content-Type"] = "application/json; charset=utf-8; application/x-www-form-urlencoded"
	hs["X-Content-Type-Options"] = "nosniff"
	hs["Content-Security-Policy"] = "default-src 'none'"
	hs["Server"] = "lbry.com"
	hs["Access-Control-Allow-Origin"] = "*"
	api.ResponseHeaders = hs
	api.Log = func(request *http.Request, response *api.Response, err error) {
		if response.Status >= http.StatusInternalServerError {
			logrus.Error(err)
		}
		if err != nil {
			logrus.Debug("Error: ", err)
		}
		consoleText := request.RemoteAddr + " [" + strconv.Itoa(response.Status) + "]: " + request.Method + " " + request.URL.Path
		logrus.Debug(color.GreenString(consoleText))

	}
	//API Chainquery DB connection
	chainqueryInstance, err := db.InitAPIQuery(config.GetAPIMySQLDSN(), false)
	if err != nil {
		logrus.Panic("unable to connect to chainquery database instance for API Server: ", err)
	}
	defer db.CloseDB(chainqueryInstance)
	router := sw.NewRouter()

	logrus.Fatal(http.ListenAndServe(hostAndPort, router))
}
