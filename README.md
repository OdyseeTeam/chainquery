
# LBRY Chainquery

[![Build Status](https://travis-ci.org/lbryio/chainquery.svg?branch=master)](https://travis-ci.org/lbryio/chainquery)

![Code Climate](https://img.shields.io/codeclimate/tech-debt/lbryio/chainquery.svg)

[![Go Report Card](https://goreportcard.com/badge/github.com/lbryio/chainquery)](https://goreportcard.com/report/github.com/lbryio/chainquery)

[![Maintainability](https://api.codeclimate.com/v1/badges/3f60ac6b2d7e96f93992/maintainability)](https://codeclimate.com/github/lbryio/chainquery/maintainability)

![GitHub release](https://img.shields.io/github/release/lbryio/chainquery.svg)

![Github commits (since latest release)](https://img.shields.io/github/commits-since/lbryio/chainquery/latest.svg)

[![Coverage Status](https://coveralls.io/repos/github/lbryio/chainquery/badge.svg?branch=master)](https://coveralls.io/github/lbryio/chainquery?branch=master)

## Prerequisites

### OS Specifics

#### OSX

- In order to use  `wget` you will need `brew install wget` (used in [build.sh](/scripts/build.sh))
- Chainquery is built for Linux by default in  [build.sh](/scripts/build.sh), so you will need to modify the cross compilation for an OSX build.
- Be sure to give execute privileges to the [scripts](/scripts) you plan to use.

### Go

Make sure you have Go 1.10+ (required for [go-releaser](https://goreleaser.com/))

- Ubuntu: https://launchpad.net/~longsleep/+archive/ubuntu/golang-backports or https://github.com/golang/go/wiki/Ubuntu
- OSX: `brew install go`

### MySQL

- Install and run mysql.(OSX: `brew install mysql`)
- Create `chainquery` database.
- Create user `lbry` with password `lbry` and grant it all permissions on `chainquery` db.

### Lbrycrd


- Install lbrycrdd (https://github.com/lbryio/lbrycrd/releases)
- Ensure `~/.lbrycrd/lbrycrd.conf` file exists with username and password.
  If you don't have one, run:

  ```
  mkdir -p ~/.lbrycrd
  echo -e "rpcuser=lbryrpc\nrpcpassword=$(env LC_CTYPE=C LC_ALL=C tr -dc A-Za-z0-9 < /dev/urandom | head -c 16 | xargs)" > ~/.lbrycrd/lbrycrd.conf
  ```

- Run `./lbrycrdd -server -daemon -txindex -conf=$HOME/.lbrycrd/lbrycrd.conf`. If you get an error about indexing, add the `-reindex` flag for one run. You will only need to
  reindex once.

## Configuration

Chainquery can be [configured](/config/default/chainqueryconfig.toml) via toml file.

## Running from Source

```
go get -u github.com/lbryio/chainquery
cd "$(go env GOPATH)/src/github.com/lbryio/chainquery"
./dev.sh
```

## The Model 

The model of Chainquery at its foundation consist of the fundamental data types found in the block chain.
This information is then expounded on with additional columns and tables that make querying the data much easier.

### [Latest Schema](/db/chainquery_schema.sql)

## What does Chainquery consist of?

Chainquery consists of 4 main parts. The API Server, the Daemon, the Job Scheduler, and the upgrade manager. 

### API Server

The API Server services either structured queries via defined APIs or raw SQL against 
the Chainquery MySQL database. The APIs are documented via [Chainquery APIs](https://lbryio.github.io/chainquery/),
a work in progress :) . 

### Daemon

The Daemon is responsible for updating the Chainquery database to keep it in sync with lbrycrd data. The daemon runs periodically to check if there are newly 
confirmed(6 confirmations currently) blocks that need to be processed. The Daemon simply processes the block and its
transactions. The entry points are [daemon iterations](/daemon/daemon.go)(`func daemonIteration()`) [block processing](/daemon/processing/block.go)(`func RunBlockProcessing(height *uint64)`), 
[transaction processing](/daemon/processing/transaction.go)(`func ProcessTx(jsonTx *lbrycrd.TxRawResult, blockTime uint64)`).

### Job Scheduler

The job scheduler schedules different types of jobs to update the Chainquery database [example](/daemon/jobs/claimtriesync.go).
These jobs synchronize different areas of the data either to make queries faster or ascertain information that is not
directly part of the raw blockchain. The example provided is leveraged to handle the status of a claim which is actually
stored in the ClaimTrie of LBRYcrd. So it runs periodically to make sure Chainquery has the most up to date status of 
claims in the trie. The table `job_status` stores the current state of a particular job, like when it last synced.

### Upgrade Manager

The upgrade manager handles data upgrades between versions. The table  `application_status` stores information about the
state of the application as it relates to the data, api and app versions. This is all leveraged by the upgrade manager so it 
knows what scripts might need to be run to keep the data in sync across deployments. The [scripts](/daemon/upgrademanager/script.go)
are foundation of the [upgrade manager](/daemon/upgrademanager/upgrade.go).

## Contributing

Contributions to this project are welcome, encouraged, and compensated. For more details, see [lbry.io/faq/contributing](https://lbry.io/faq/contributing)

The `master` branch is regularly built and tested, but is not guaranteed to be
completely stable. [Releases](https://github.com/lbryio/chainquery/releases) are created
regularly to indicate new official, stable release versions.

Developers are strongly encouraged to write unit tests for new code, and to
submit new unit tests for old code. Unit tests can be compiled and run
 with: `go test ./...` from the source directory which should be `$GOPATH/github.com/lbryio/chainquery`.

## Updating the generated models

We use [sqlboiler](https://github.com/lbryio/sqlboiler) to generate our data models based on the db schema. If you make  schema changes, run `./gen_models.sh` to
regenerate the models.

**A note of caution:** the models are generated by connecting to the MySQL server and inspecting the current schema. If you made any db schema changes by hand, then the
schema may be out of sync with the migrations. Here's the safe way to ensure that the models match the migrations:

- Put all the schema changes you want to make into a migration.
- In mysql, drop and recreate the db you're using, so that it's empty.
- Run `./dev.sh`. This will run all the migrations on the empty db.
- Run `./gen_models.sh` to update the models.

This process ensures that the generated models will match the updated schema exactly, so there are no surprises when the migrations are applied to the live db.

## License

This project is MIT licensed. For the full license, see [LICENSE](LICENSE).

## Security

We take security seriously. Please contact security@lbry.io regarding any security issues.
Our PGP key is [here](https://keybase.io/lbry/key.asc) if you need it.

## Contact

The primary contact for this project is [@tiger5226](https://github.com/tiger5226) (beamer@lbry.io)
