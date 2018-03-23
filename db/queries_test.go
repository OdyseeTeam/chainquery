package db

import (
	"testing"

	"github.com/lbryio/chainquery/model"

	"github.com/volatiletech/sqlboiler/queries/qm"
)

func TestQueryGetAddressSummary(t *testing.T) {
	//Need to add setup here so it can connect to the db
	addresses, err := model.AddressesG(qm.Limit(1000)).All()
	if err != nil {
		t.Error(err)
	}

	for i := range addresses {
		address := addresses[i]
		_, err := GetAddressSummary(address.Address)
		if err != nil {
			t.Error(err)
		}

	}
}
