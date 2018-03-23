package db

import (
	"github.com/volatiletech/sqlboiler/queries"
)

type AddressSummary struct {
	Id            uint64  `boil:id`
	Address       string  `boil:address`
	TotalReceived float64 `boil:total_received`
	TotalSent     float64 `boil:total_sent`
	Balance       float64 `boil:balance`
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
