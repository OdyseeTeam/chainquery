package db

import (
	"github.com/lbryio/chainquery/migration"
	"github.com/lbryio/lbry.go/errors"

	_ "github.com/go-sql-driver/mysql" // import mysql
	"github.com/jmoiron/sqlx"
	_ "github.com/jteeuwen/go-bindata" // so it's detected by `dep ensure`
	"github.com/rubenv/sql-migrate"
	log "github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
)

func Init(dsn string, debug bool) (*QueryLogger, error) {
	dsn += "?parseTime=1&collation=utf8mb4_unicode_ci"
	dbConn, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, errors.Err(err)
	}

	err = dbConn.Ping()
	if err != nil {
		return nil, errors.Err(err)
	}

	logWrapper := &QueryLogger{DB: dbConn}
	if debug {
		logWrapper.Logger = log.StandardLogger()
		boil.DebugMode = true // this just prints everything twice
	}

	boil.SetDB(logWrapper)

	// ensure that db supports transactions
	_, ok := boil.GetDB().(boil.Beginner)
	if !ok {
		//return nil, errors.Err("database does not support transactions")
	}

	migrations := &migrate.AssetMigrationSource{
		Asset:    migration.Asset,
		AssetDir: migration.AssetDir,
		Dir:      "migration",
	}
	n, migrationErr := migrate.Exec(dbConn.DB, "mysql", migrations, migrate.Up)
	if migrationErr != nil {
		return nil, errors.Err(migrationErr)
	}
	log.Printf("Applied %d migrations", n)

	return logWrapper, nil
}
