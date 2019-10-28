package db

import (
	"github.com/lbryio/chainquery/migration"
	"github.com/lbryio/lbry.go/extras/errors"

	_ "github.com/go-sql-driver/mysql" // import mysql
	"github.com/jmoiron/sqlx"
	_ "github.com/jteeuwen/go-bindata" // so it's detected by `dep ensure`
	migrate "github.com/rubenv/sql-migrate"
	log "github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
)

// Init initializes a database connection based on the dsn provided. It also sets it as the global db connection.
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
		boil.DebugMode = true
	}

	boil.SetDB(logWrapper)

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

func dbInitConnection(dsn string, driverName string, debug bool) (*sqlx.DB, *QueryLogger, error) {
	dsn += "?parseTime=1&collation=utf8mb4_unicode_ci"
	dbConn, err := sqlx.Connect(driverName, dsn)
	if err != nil {
		return nil, nil, errors.Err(err)
	}

	err = dbConn.Ping()
	if err != nil {
		return nil, nil, errors.Err(err)
	}

	logWrapper := &QueryLogger{DB: dbConn}
	if debug {
		logWrapper.Logger = log.StandardLogger()
		//boil.DebugMode = true // this just prints everything twice
	}

	return dbConn, logWrapper, nil
}

// CloseDB is a wrapper function to allow error handle when it is usually deferred.
func CloseDB(db *QueryLogger) {
	if err := db.Close(); err != nil {
		log.Error("Closing DB Error: ", err)
	}
}
