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
	err := queries.RawG(`CALL address_summary(?)`, address).Bind(&addressSummary)
	if err != nil {
		return nil, err
	}

	return &addressSummary, nil

}
