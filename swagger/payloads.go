package swagger

import (
	"github.com/lbryio/chainquery/db"
	"github.com/sirupsen/logrus"
	"net/http"
)

func HandleAction(operation string, w http.ResponseWriter, r http.Request) error {
	return nil
}

func GetStatusPayload() (interface{}, error) {
	status, err := db.GetTableStatus()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return status, nil

}
