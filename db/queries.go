package db

import (
	g "github.com/lbryio/chainquery/swagger/clients/goclient"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries"
)

type AddressSummary struct {
	Id            uint64  `boil:id`
	Address       string  `boil:address`
	TotalReceived float64 `boil:total_received`
	TotalSent     float64 `boil:total_sent`
	Balance       float64 `boil:balance`
}

func GetTableStatus() (*g.TableStatus, error) {
	println("here2")
	stats := g.TableStatus{}
	rows, err := boil.GetDB().Query(
		`SELECT TABLE_NAME as "table",` +
			`SUM(TABLE_ROWS) as "rows" ` +
			`FROM INFORMATION_SCHEMA.TABLES ` +
			`WHERE TABLE_SCHEMA = "lbrycrd" ` +
			`GROUP BY TABLE_NAME;`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var statrows = []g.TableSize{}
	for rows.Next() {
		var stat g.TableSize
		err = rows.Scan(&stat.TableName, &stat.NrRows)
		if err != nil {
			return nil, err
		}
		statrows = append(statrows, stat)
	}

	stats.Status = statrows

	return &stats, nil
}

func GetAddressSummary(address string) (*AddressSummary, error) {
	addressSummary := AddressSummary{}
	err := queries.RawG(
		`SELECT address.address, `+
			`SUM(ta.credit_amount) AS total_received, `+
			`SUM(ta.debit_amount) AS total_sent,`+
			`(SUM(ta.credit_amount) - SUM(ta.debit_amount)) AS balance `+
			`FROM address LEFT JOIN transaction_address as ta ON ta.address_id = address.id `+
			`WHERE address.address=? `+
			`GROUP BY address.address `, address).Bind(&addressSummary)

	if err != nil {
		return nil, err
	}

	return &addressSummary, nil

}
