
# LBRY Chainquery

[![CI](https://github.com/lbryio/chainquery/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/lbryio/chainquery/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/lbryio/chainquery)](https://goreportcard.com/report/github.com/lbryio/chainquery)
[![Maintainability](https://api.codeclimate.com/v1/badges/3f60ac6b2d7e96f93992/maintainability)](https://codeclimate.com/github/lbryio/chainquery/maintainability)
![GitHub release](https://img.shields.io/github/release/lbryio/chainquery.svg)
![Github commits (since latest release)](https://img.shields.io/github/commits-since/lbryio/chainquery/latest.svg)

Chainquery reads the LBRY blockchain into a structured **MySQL** database so that
blocks, transactions, addresses, and the full LBRY **claim** system (channels,
publishes, supports, tips/purchases, tags) can be queried quickly with plain SQL
instead of walking the chain over RPC.

It connects to a LBRY full node over JSON-RPC — **lbcd** in production (LBRY's
Go full-node implementation, which the stack has migrated to), or the original
**lbrycrd** — ingests blocks as they are mined, keeps the database in sync
(including across blockchain reorganizations), computes derived data via
scheduled jobs, and exposes an HTTP API — including a public, read-only SQL
endpoint.

## What Chainquery consists of

Everything runs from a single binary (`chainquery serve`), which starts the API
server and the daemon together. There are four logical parts.

### 1. Daemon

The daemon (`daemon/daemon.go`, `daemon/processing/`) keeps the database in sync
with lbrycrd. On each iteration it asks lbrycrd for the current height and
processes every block above the last height it stored.

- **Blocks are processed strictly in order.** A channel handshake
  (`blockQueue` → `blockProcessedChan`) plus a global `BlockLock` mutex guarantee
  block N+1 never starts before N is committed; out-of-order processing panics by
  design to avoid corrupting the data.
- **Transactions within a block are processed in parallel.** A worker pool
  (`maxparalleltxprocessing`, default `NumCPU`) drains a job queue. Because a
  transaction can spend an output created by another transaction in the same
  block, failed transactions are pushed onto a redo queue and retried up to
  `maxfailures` times (default 1000). Transactions are pre-sorted by dependency
  depth (`optimizeOrderToProcess`) so parents tend to run before children.
- **Reorgs are handled automatically.** Before processing a block, the daemon
  compares the stored previous-block hash against the chain. On a mismatch it
  recursively deletes diverged blocks (up to depth 100), logs the reorg depth,
  and reprocesses from the divergence height.
- **Processing modes** control throttling (`daemonmode`): beast (0, no delay),
  slow-and-steady (1, 100ms/block), delay (2, configurable), and daemon (3,
  one block per daemon iteration).

Entry points: `daemonIteration()`, `processing.RunBlockProcessing(stopper, height)`,
`processing.ProcessTx(...)`.

### 2. API Server

The API server (`swagger/apiserver/`, handlers in `apiactions/`) binds to
`apihostport` (default `0.0.0.0:6300`) and uses a **separate** read DSN
(`apimysqldsn`). Routes (`swagger/apiserver/go/routers.go`):

| Method | Path                  | Description                                                        | Auth          |
|--------|-----------------------|--------------------------------------------------------------------|---------------|
| GET    | `/api/`               | Index — returns `Hello World!`                                     | none          |
| GET    | `/api/sql`            | **Public read-only SQL.** Runs the `query` param against MySQL with an injected `MAX_EXECUTION_TIME` and a `maxsqlapitimeout` cap | none |
| GET    | `/api/addresssummary` | Address received / spent / balance                                 | none          |
| GET    | `/api/status`         | Table names and sizes                                              | none          |
| GET    | `/api/validate`       | Validate chain data                                               | none          |
| GET    | `/api/process`        | Process a block or range of blocks                                | API key       |
| GET    | `/api/sync/name`      | Re-sync claimtrie state for a claim name                          | API key       |
| GET    | `/api/sync/addresses` | Sync address balances                                             | API key       |
| GET    | `/api/sync/txvalues`  | Sync transaction values                                           | API key       |
| GET    | `/metrics`            | Prometheus metrics                                                | basic auth    |

API-key endpoints are rejected unless the supplied `Key` is listed in the
`apikeys` config (empty by default = disabled). Prometheus `/metrics` is guarded
by `promuser`/`prompass` when set.

API docs (Swagger, WIP): https://lbryio.github.io/chainquery/ — spec lives at
[`swagger/apiserver/api/swagger.yaml`](/swagger/apiserver/api/swagger.yaml).

### 3. Job Scheduler

Periodic jobs (`daemon/jobs/`) compute data that isn't directly part of the raw
blockchain, or that is faster to precompute. Scheduled in `initJobs()`:

| Job                       | Interval | Purpose                                                  |
|---------------------------|----------|----------------------------------------------------------|
| Claimtrie Sync            | 15m      | Claim status/effective amount from lbrycrd's ClaimTrie   |
| Mempool Sync              | 1s       | Unconfirmed transactions                                 |
| Certificate Sync          | 5s       | Channel (certificate) data                               |
| Chain Sync                | 5s       | Chain-derived data (must run < 2.5m, see code note)      |
| Validate Chain            | 24h      | Integrity validation                                     |
| Address Balance Sync      | 24h      | Recompute address balances                               |
| Transaction Value Sync    | 24h      | Recompute transaction values                             |
| Claim Count in Channel    | 24h      | Number of claims per channel                             |

Jobs can also be run one-off via `chainquery run <job>` (see CLI below). The
`job_status` table records each job's last run.

### 4. Upgrade Manager

`daemon/upgrademanager/` runs data/schema upgrade scripts between versions on
startup. The `application_status` table tracks the app/data/api versions so the
manager knows which scripts to apply.

## Data Model

The schema is the fundamental blockchain types — `block`, `transaction`,
`input`, `output`, `address` — enriched with the LBRY claim system: `claim`,
`support`, `purchase`, `tag`, `claim_tag`, `claim_in_list`, `abnormal_claim`,
plus bookkeeping tables (`job_status`, `application_status`). Models in
[`model/`](/model) are generated by [SQLBoiler](https://github.com/volatiletech/sqlboiler).

- **Reference schema:** [`db/chainquery_schema.sql`](/db/chainquery_schema.sql)

## Notifications

Chainquery can emit real-time events:

- **Sockety** (`socketyurl` / `socketytoken`) — a `new_block` notification is
  sent on every processed block.
- **Subscribers** (`config`) — webhook URLs for `payment` and `new_claim` events.
- **Slack** (`slackbottoken`, `slackchannel`, `slackloglevel`) — Slack app log
  forwarding via `chat.postMessage`, including reorg depth warnings.

## Prerequisites

- **Go** — the module targets Go **1.26+** (see [`go.mod`](/go.mod)).
- **MySQL 8** — create a `chainquery` database and a user that can access it. The
  default DSN expects user `chainquery` / password `chainquery`:

  ```sql
  CREATE DATABASE IF NOT EXISTS chainquery;
  CREATE USER 'chainquery'@'localhost' IDENTIFIED BY 'chainquery';
  GRANT ALL ON chainquery.* TO 'chainquery'@'localhost';
  ```

  MySQL must allow stored functions: set `log_bin_trust_function_creators = 1`.

- **A LBRY full node** — in production this is [**lbcd**](https://github.com/lbryio/lbcd),
  LBRY's Go node that the stack has migrated to. The original
  [lbrycrd](https://github.com/lbryio/lbrycrd) still works, because chainquery
  only speaks the node's JSON-RPC and doesn't care which implementation answers.
  - Enable a transaction index (`txindex=1` for lbcd, `-txindex` for lbrycrdd) —
    chainquery relies on `getrawtransaction`.
  - Both default their RPC to port **9245**, matching chainquery's default
    `lbrycrdurl` (`rpc://lbry:lbry@localhost:9245`). Set the node's
    `rpcuser`/`rpcpass` (in `~/.lbcd/lbcd.conf` or `~/.lbrycrd/lbrycrd.conf`) to
    match, or override `lbrycrdurl`.
  - lbcd enables RPC TLS by default; chainquery connects lbrycrd-style, so use
    lbcd's `notls=1` for a local plaintext RPC.
  - The config key (`lbrycrdurl`) and the Go package (`lbrycrd/`) keep the
    historical name regardless of which node you actually run.

## Configuration

Chainquery is configured via a TOML file. The annotated default is
[`config/default/chainqueryconfig.toml`](/config/default/chainqueryconfig.toml).
Resolution order: `$HOME`, the working directory, then the in-repo default
(override with `--configpath`).

Common settings:

| Key                       | Default                                               | Purpose                                          |
|---------------------------|-------------------------------------------------------|--------------------------------------------------|
| `lbrycrdurl`              | `rpc://lbry:lbry@localhost:9245`                      | LBRY full-node RPC (lbcd in prod, or lbrycrd)    |
| `mysqldsn`                | `chainquery:chainquery@tcp(localhost:3306)/chainquery`| Daemon (read/write) DB                           |
| `apimysqldsn`             | same as `mysqldsn`                                    | API server (read) DB                             |
| `apihostport`             | `0.0.0.0:6300`                                         | API bind address                                 |
| `blockchainname`          | `lbrycrd_main`                                        | Chain params (`_main` / `_testnet` / `_regtest`) |
| `daemonmode`              | `0`                                                   | Processing throttle mode                         |
| `maxfailures`             | `1000`                                                | Per-transaction retries before block rollback    |
| `maxparalleltxprocessing` | `NumCPU`                                              | Tx worker count per block                        |
| `maxsqlapitimeout`        | `5`                                                   | Max seconds for `/api/sql` queries               |
| `apikeys`                 | `[]`                                                  | Keys allowed to call authorized endpoints        |

## Building and running

The repository ships a prebuilt Linux binary in [`bin/`](/bin), but to build
from source:

```bash
git clone https://github.com/lbryio/chainquery.git
cd chainquery
go build -o chainquery .
```

`scripts/build.sh` produces a static (`CGO_ENABLED=0`) Linux release binary at
`bin/chainquery` with version metadata baked in.

Initialize the database schema (runs the migrations), then start the service:

```bash
./chainquery serve db   # create/upgrade the schema
./chainquery serve      # run the daemon + API server
```

`-help` lists all flags. `-d`/`--debug` and `-t`/`--trace` raise log verbosity.

### CLI commands

| Command                  | Description                                                              |
|--------------------------|--------------------------------------------------------------------------|
| `chainquery serve`       | Run the daemon and API server (the main mode)                            |
| `chainquery serve db`    | Create/upgrade the database schema and exit                              |
| `chainquery run <job>`   | Run a single job: `claimcount`, `claimtrie`, `certificate`, `mempool`, `transactionvalue`, `chain`, `outputfix` |
| `chainquery version`     | Print version information                                                |

## Development

For an auto-reloading dev loop, [`dev.sh`](/dev.sh) uses
[`reflex`](https://github.com/cespare/reflex) to rebuild and run
`go generate && go run *.go serve` on every `.go` change. It detects a running
`lbrycrdd` and connects via its conf file.

Run the unit tests:

```bash
go test -race ./...
```

There is an end-to-end test that spins up lbrycrd in Docker
([`e2e/e2e.sh`](/e2e/e2e.sh)); it requires Docker. Note the e2e helper still
invokes the legacy `docker-compose` (v1) CLI.

### Continuous integration

GitHub Actions runs CI from [`.github/workflows/ci.yml`](/.github/workflows/ci.yml).
The workflow:

1. Starts a MySQL 8 service.
2. Runs `go mod download` and `go mod verify`.
3. Runs `go test -race ./...`.
4. Builds `bin/chainquery` with [`scripts/build.sh`](/scripts/build.sh).
5. Checks that generated migration bindata did not drift.
6. Uploads the Linux binary artifact.

The workflow uses Go `1.26.4`. The test job runs in the official
`golang:1.26.4-bookworm` container, and the build/release jobs use GitHub's
hosted `ubuntu-24.04` runner. Build tools are pinned in `scripts/build.sh` so CI
does not silently upgrade to a newer Go toolchain.

Pushing a tag that matches `v*` publishes a GitHub Release. The release contains
only the Linux amd64 binary:

- `chainquery-linux-amd64`
- `chainquery-linux-amd64.sha256`

The module depends on `github.com/OdyseeTeam/sockety`. If the default
`GITHUB_TOKEN` cannot read that private module, configure a repository secret
named `GO_MODULES_TOKEN` with access to it.

Local workflow runs use [`act`](https://github.com/nektos/act):

```bash
act pull_request \
  -W .github/workflows/ci.yml \
  -P ubuntu-24.04=catthehacker/ubuntu:act-24.04 \
  --secret-file /path/to/act.secrets
```

The `catthehacker/ubuntu:act-24.04` image is only the local `act` runner image
for the GitHub `ubuntu-24.04` runner label.

`actions/upload-artifact@v7` is skipped only under `act`, because current `act`
artifact emulation does not support the v7 upload protocol. The upload step runs
on GitHub Actions.

### Updating the generated models

Models are generated with [SQLBoiler](https://github.com/volatiletech/sqlboiler)
by introspecting the live MySQL schema, via [`scripts/gen_models.sh`](/scripts/gen_models.sh).
Because it reads the actual database, the safe way to keep models in sync with
migrations is:

1. Put schema changes into a migration under [`migration/`](/migration).
2. Drop and recreate the database so it's empty.
3. Run `chainquery serve db` to apply all migrations.
4. Run `scripts/gen_models.sh` to regenerate the models.

This guarantees the models match what the migrations actually produce.

## Contributing

Contributions are welcome, encouraged, and compensated. See
[https://lbry.tech/contribute](https://lbry.tech/contribute).

`master` is regularly built and tested but is not guaranteed stable.
[Releases](https://github.com/lbryio/chainquery/releases) mark official stable
versions. Please write unit tests for new code.

## License

This project is MIT licensed. For the full license, see [LICENSE](LICENSE).

## Security

We take security seriously. Please contact security@lbry.io regarding any
security issues. Our PGP key is [here](https://lbry.com/faq/pgp-key) if you need it.

## Contact

The primary contact for this project is [@tiger5226](https://github.com/tiger5226)
(beamer -at- odysee.com).
