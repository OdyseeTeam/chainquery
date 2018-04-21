package apis

import (
	"net/http"

	"github.com/lbryio/chainquery/db"
	"github.com/lbryio/lbry.go/errors"

	"github.com/sirupsen/logrus"
)

const (
	ADDRESSSUMARY = "AddressSummary"
	STATUS        = "Status"
)

// Response is returned by API handlers
type Response struct {
	TimeSpent   string
	Status      int
	Data        interface{}
	RedirectURL string
	Error       error
}

func HandleAction(operation string, w http.ResponseWriter, r *http.Request) (*Response, error) {
	switch operation {
	case STATUS:
		payload, err := getStatusPayload(r)
		response := processPayload(payload, err)
		return response, nil
	case ADDRESSSUMARY:
		payload, err := getAddressSummary(r)
		response := processPayload(payload, err)
		return response, nil
	default:
		return nil, errors.Base(operation + " is not implmented yet.")
	}
}

func processPayload(payload interface{}, error error) *Response {
	response := Response{}
	if error != nil {
		response.Error = error
		return &response
	}
	response.Status = http.StatusOK
	response.Data = payload

	return &response
}

func getStatusPayload(r *http.Request) (interface{}, error) {
	status, err := db.GetTableStatus()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return status, nil
}

func getAddressSummary(r *http.Request) (interface{}, error) {
	address := r.FormValue("Address")
	summary, err := db.GetAddressSummary(address)
	if err != nil {
		return nil, err
	}
	return summary, nil
}
