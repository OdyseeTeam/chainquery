package db

import (
	g "github.com/lbryio/chainquery/swagger/clients/goclient"
	"github.com/lbryio/chainquery/util"

	"github.com/lbryio/chainquery/meta"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries"
)

// AddressSummary summarizes information for an address from chainquery database
type AddressSummary struct {
	ID            uint64  `boil:"id"`
	Address       string  `boil:"address"`
	TotalReceived float64 `boil:"total_received"`
	TotalSent     float64 `boil:"total_sent"`
	Balance       float64 `boil:"balance"`
}

// GetTableStatus provides size information for the tables in the chainquery database
func GetTableStatus() (*g.ChainqueryStatus, error) {
	stats := g.ChainqueryStatus{}
	rows, err := boil.GetDB().Query(
		`SELECT TABLE_NAME as "table",` +
			`SUM(TABLE_ROWS) as "rows" ` +
			`FROM INFORMATION_SCHEMA.TABLES ` +
			`WHERE TABLE_SCHEMA = "chainquery" ` +
			`GROUP BY TABLE_NAME;`)

	if err != nil {
		return nil, err
	}
	defer util.CloseRows(rows)
	var statrows []g.TableSize
	for rows.Next() {
		var stat g.TableSize
		err = rows.Scan(&stat.TableName, &stat.NrRows)
		if err != nil {
			return nil, err
		}
		statrows = append(statrows, stat)
	}

	stats.TableStatus = statrows
	stats.SemVersion = meta.GetSemVersion()
	stats.VersionShort = meta.GetVersion()
	stats.VersionLong = meta.GetVersionLong()
	stats.CommitMessage = meta.GetCommitMessage()

	return &stats, nil
}

// GetAddressSummary returns summary information of an address in the chainquery database.
func GetAddressSummary(address string) (*AddressSummary, error) {
	addressSummary := AddressSummary{}
	err := queries.Raw(
		`SELECT address.address, `+
			`SUM(ta.credit_amount) AS total_received, `+
			`SUM(ta.debit_amount) AS total_sent,`+
			`(SUM(ta.credit_amount) - SUM(ta.debit_amount)) AS balance `+
			`FROM address LEFT JOIN transaction_address as ta ON ta.address_id = address.id `+
			`WHERE address.address=? `+
			`GROUP BY address.address `, address).BindG(nil, &addressSummary)

	if err != nil {
		return nil, err
	}

	return &addressSummary, nil

}

// APIQuery is the entry point from the API to chainquery. The results are turned into json.
func APIQuery(query string, args ...interface{}) (interface{}, error) {
	rows, err := apiQuery(query, args...)
	if err != nil {
		return nil, err
	}
	util.CloseRows(rows)
	return jsonify(rows), nil

}
