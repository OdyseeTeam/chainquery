package db

import (
	"database/sql"
	"strconv"

	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/volatiletech/sqlboiler/boil"
)

//Executor used for chainquery db
var chainquery boil.Executor

// InitAPIQuery initializes the api chainquery db connection
func InitAPIQuery(dsn string, debug bool) (*QueryLogger, error) {
	if dsn == "" {
		return nil, errors.Base("chainquery DSN was not provided.")
	}
	_, logWrapper, err := dbInitConnection(dsn, "mysql", debug)
	if err != nil {
		return nil, err
	}

	//Set chainquery global executor
	chainquery = logWrapper

	return logWrapper, nil
}

// Used to query the chainquery database...
func apiQuery(query string, args ...interface{}) (*sql.Rows, error) {
	if chainquery == nil {
		return nil, errors.Base("no connection to chainquery database.")
	}
	return chainquery.Query(query, args...)
}

type rowSlice []*map[string]interface{}

func jsonify(rows *sql.Rows) rowSlice {
	columns, err := rows.Columns()
	if err != nil {
		panic(err.Error())
	}

	values := make([]interface{}, len(columns))

	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	c := 0

	data := rowSlice{}

	for rows.Next() {
		results := make(map[string]interface{})
		err = rows.Scan(scanArgs...)
		if err != nil {
			panic(err.Error())
		}

		for i, value := range values {
			switch value := value.(type) {
			case nil:
				results[columns[i]] = nil

			case []byte:
				s := string(value)
				x, err := strconv.Atoi(s)

				if err != nil {
					results[columns[i]] = s
				} else {
					results[columns[i]] = x
				}

			default:
				results[columns[i]] = value
			}
		}
		data = append(data, &results)
		c++
	}

	return data
}
