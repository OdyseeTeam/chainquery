package main

import (
	"github.com/lbryio/chainquery/swagger/clients/go"
	"testing"
)

func TestAPIStatus(t *testing.T) {
	println("Here")
	api := swagger.NewDefaultApiWithBasePath("http://localhost:8080/v1")
	tableStatus, response, err := api.Status()
	if err != nil {
		t.Error(err)
	}
	println("ResponseStatus: ", response.Status)
	for _, table := range tableStatus.Status {
		println("Table: ", table.TableName, " Rows: ", table.NrRows)
	}
}
