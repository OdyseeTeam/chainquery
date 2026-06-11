package db

import (
	"time"

	"github.com/lbryio/chainquery/migration"
	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/kevinburke/go-bindata/v4" // so it's detected by `dep ensure`
	migrate "github.com/rubenv/sql-migrate"
	log "github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

var maxOpenConns = 50
var maxIdleConns = 10
var connMaxLifetime = 5 * time.Minute
var connectTimeout = 20 * time.Second
var readTimeout = 2 * time.Minute
var writeTimeout = 2 * time.Minute

func ConfigureConnection(maxOpen int, maxIdle int, maxLifetime time.Duration, dialTimeout time.Duration, read time.Duration, write time.Duration) {
	maxOpenConns = maxOpen
	maxIdleConns = maxIdle
	connMaxLifetime = maxLifetime
	connectTimeout = dialTimeout
	readTimeout = read
	writeTimeout = write
}

// Init initializes a database connection based on the dsn provided. It also sets it as the global db connection.
func Init(dsn string, debug bool) (*QueryLogger, error) {
	dsn, err := prepareDSN(dsn)
	if err != nil {
		return nil, errors.Err(err)
	}
	dbConn, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, errors.Err(err)
	}

	err = dbConn.Ping()
	if err != nil {
		return nil, errors.Err(err)
	}
	applyPoolSettings(dbConn)

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
	var err error
	dsn, err = prepareDSN(dsn)
	if err != nil {
		return nil, nil, errors.Err(err)
	}
	dbConn, err := sqlx.Connect(driverName, dsn)
	if err != nil {
		return nil, nil, errors.Err(err)
	}

	err = dbConn.Ping()
	if err != nil {
		return nil, nil, errors.Err(err)
	}
	applyPoolSettings(dbConn)

	logWrapper := &QueryLogger{DB: dbConn}
	if debug {
		logWrapper.Logger = log.StandardLogger()
		//boil.DebugMode = true // this just prints everything twice
	}

	return dbConn, logWrapper, nil
}

func prepareDSN(dsn string) (string, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", errors.Err(err)
	}
	cfg.ParseTime = true
	if cfg.Collation == "" {
		cfg.Collation = "utf8mb4_unicode_ci"
	}
	if cfg.Loc == nil {
		cfg.Loc = time.Local
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = connectTimeout
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = readTimeout
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = writeTimeout
	}
	return cfg.FormatDSN(), nil
}

func applyPoolSettings(dbConn *sqlx.DB) {
	if maxOpenConns > 0 {
		dbConn.SetMaxOpenConns(maxOpenConns)
	}
	if maxIdleConns > 0 {
		dbConn.SetMaxIdleConns(maxIdleConns)
	}
	if connMaxLifetime > 0 {
		dbConn.SetConnMaxLifetime(connMaxLifetime)
	}
}

// CloseDB is a wrapper function to allow error handle when it is usually deferred.
func CloseDB(db *QueryLogger) {
	if err := db.Close(); err != nil {
		log.Error("Closing DB Error: ", err)
	}
}
