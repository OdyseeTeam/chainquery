package db

import (
	"os"
	"testing"

	"github.com/lbryio/chainquery/config"
	"github.com/lbryio/chainquery/model"

	"github.com/volatiletech/sqlboiler/queries/qm"
)

func TestMain(m *testing.M) {
	config.InitializeConfiguration()
	dbInstance, err := Init(config.GetMySQLDSN(), false)
	if err != nil {
		panic(err)
	}
	defer CloseDB(dbInstance)
	os.Exit(m.Run())
}

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

func TestQueryGetTableStatus(t *testing.T) {
	stats, err := GetTableStatus()
	if err != nil {
		t.Error(err)
	}
	println("TableName     NrRows", len(stats.Status))
	for _, stat := range stats.Status {
		println(stat.TableName, ":", stat.NrRows)
	}
}
