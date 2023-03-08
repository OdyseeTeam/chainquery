package db

import (
	"os"
	"testing"

	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

func TestMain(m *testing.M) {
	dbInstance, err := Init("chainquery:chainquery@tcp(localhost:3306)/chainquery", false)
	if err != nil {
		panic(errors.FullTrace(err))
	}
	defer CloseDB(dbInstance)
	os.Exit(m.Run())
}

func TestQueryGetAddressSummary(t *testing.T) {
	//Need to add setup here so it can connect to the db
	addresses, err := model.Addresses(qm.Limit(1000)).AllG()
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
	println("TableName     NrRows", len(stats.TableStatus))
	for _, stat := range stats.TableStatus {
		println(stat.TableName, ":", stat.NrRows)
	}
}
