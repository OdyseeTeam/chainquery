package jobs

import (
	"time"

	"github.com/volatiletech/null"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
)

var start = time.Date(2017, 01, 01, 0, 0, 0, 0, time.UTC)
var days = 1269
var nrPublishes = int64(9)

type nthPublisher struct {
	publishers map[string]int64
	date       time.Time
}

var publishers []nthPublisher

func Queries() {
	logrus.Info("hello")
	previousResult := make(map[string]int64)
	for i := 0; i < days; i++ {
		queryDate := start.Add(time.Duration(i) * 24 * time.Hour)
		rows, err := boil.GetDB().Query(`
				SELECT p.claim_id, COUNT(c.claim_id)
				FROM block b
				INNER JOIN transaction t ON  b.hash = t.block_hash_id
				INNER JOIN claim c ON c.transaction_hash_id = t.hash
				INNER JOIN claim p ON c.publisher_id = p.claim_id
				WHERE b.block_time >= ? AND b.block_time < ?
				GROUP BY p.claim_id`, queryDate.Unix(), queryDate.Add(24*time.Hour).Unix())
		defer rows.Close()
		if err != nil {
			logrus.Panic(err)
		}
		var results = copy(previousResult)
		for rows.Next() {
			var publisher null.String
			var count null.Int64
			err := rows.Scan(&publisher, &count)
			if err != nil {
				logrus.Panic(err)
			}
			if !publisher.IsZero() {
				if cnt, ok := results[publisher.String]; ok {
					results[publisher.String] = cnt + count.Int64
				} else {
					results[publisher.String] = count.Int64
				}
			}
		}
		result := nthPublisher{
			publishers: results,
			date:       queryDate,
		}
		logrus.Info("Date:", queryDate.Format("2006-01-02"), " Publishers:", len(result.publishers))
		publishers = append(publishers, result)
		previousResult = result.publishers

	}
	for _, d := range publishers {
		var publishersOverX int64
		for _, v := range d.publishers {
			if v > nrPublishes {
				publishersOverX++
			}
		}
		println(d.date.Format("2006-01-02"), ",", publishersOverX)
	}

}

func copy(toCopy map[string]int64) map[string]int64 {
	copied := make(map[string]int64)
	for k, v := range toCopy {
		copied[k] = v
	}
	return copied
}
